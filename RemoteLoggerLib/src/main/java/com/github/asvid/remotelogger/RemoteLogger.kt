package com.github.asvid.remotelogger

import android.util.Log
import kotlinx.coroutines.*
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.channels.Channel.Factory.UNLIMITED
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.io.BufferedReader
import java.io.InputStreamReader
import java.util.*
import java.util.concurrent.TimeUnit
import kotlin.math.min
import kotlin.system.exitProcess

const val DEFAULT_PORT = 3000

class RemoteLogger {
    private lateinit var config: Config
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val client = OkHttpClient.Builder()
        .pingInterval(20, TimeUnit.SECONDS)
        .build()

    private var webSocket: WebSocket? = null
    private var events: Channel<Event>? = null

    fun initialize(config: Config) {
        this.config = config
        events = Channel(UNLIMITED)
        scope.launch { readLogcatStream() }
        scope.launch { connectWithBackoff() }
        setAppCrashListener()
    }

    private suspend fun connectWithBackoff() {
        var delayMs = config.reconnectDelayMs
        var firstAttempt = true

        while (scope.isActive) {
            if (!firstAttempt) {
                delay(delayMs)
                delayMs = min(delayMs * 2, 30_000L)
            }
            firstAttempt = false

            val connected = suspendCancellableCoroutine<Boolean> { cont ->
                val request = Request.Builder()
                    .url("ws://${config.host}:${config.port}")
                    .build()

                val ws = client.newWebSocket(request, object : WebSocketListener() {
                    override fun onOpen(ws: WebSocket, response: Response) {
                        webSocket = ws
                        delayMs = config.reconnectDelayMs // reset backoff on success
                        Log.i("RemoteLogger", "Connected to ${config.host}:${config.port}")

                        if (!config.sendHistoryOnConnect) {
                            events?.tryReceive() // drain
                        }
                        scope.launch { drainQueueToWs() }
                    }

                    override fun onFailure(ws: WebSocket, t: Throwable, response: Response?) {
                        webSocket = null
                        Log.w("RemoteLogger", "WebSocket failure: ${t.message}")
                        if (cont.isActive) cont.resume(false) { }
                    }

                    override fun onClosed(ws: WebSocket, code: Int, reason: String) {
                        webSocket = null
                        Log.i("RemoteLogger", "WebSocket closed: $reason")
                        if (cont.isActive) cont.resume(false) { }
                    }
                })

                cont.invokeOnCancellation { ws.cancel() }
            }

            if (!config.reconnectOnDisconnect) break
            if (!scope.isActive) break
        }
    }

    private suspend fun drainQueueToWs() {
        events?.receiveAsFlow()?.collect { event ->
            val ws = webSocket ?: return@collect
            ws.send(event.toJson())
        }
    }

    private fun setAppCrashListener() {
        Thread.setDefaultUncaughtExceptionHandler { thread, throwable ->
            Log.e("AndroidRuntime", "--->uncaughtException:$thread<---", throwable)
            val event = Event(
                tag = "CRASH",
                message = throwable.stackTraceToString(),
                level = EventLevel.ERROR,
                time = Date().toString()
            )
            webSocket?.send(event.toJson())
            webSocket?.close(1001, "crash")
            exitProcess(1)
        }
    }

    private fun readLogcatStream() {
        cleanLogcat()

        val process = Runtime.getRuntime()
            .exec("logcat ${config.packageName} -v time")
        val reader = BufferedReader(InputStreamReader(process.inputStream))

        val logcatRegex = """(.*) ([IDWVE])(\/.+)+\ *(\( \d*\)): (.*)""".toRegex()

        while (true) {
            val line = reader.readLine() ?: break
            runCatching {
                val match = logcatRegex.findAll(line).first()
                val time = match.groupValues[1]
                val level = match.groupValues[2]
                val tag = match.groupValues[3].drop(1)
                val message = match.groupValues[5]
                events?.trySend(Event(tag, message, level.toEventLevel(), time))
            }.onFailure {
                events?.trySend(Event("LOGGER", line, EventLevel.DEBUG, Date().toString()))
            }
        }
    }

    private fun cleanLogcat() {
        ProcessBuilder()
            .command("logcat", "-c")
            .redirectErrorStream(true)
            .start()
    }
}

enum class EventLevel { INFO, ERROR, DEBUG, VERBOSE, WARNING }

@Serializable
data class Event(
    val tag: String,
    val message: String,
    val level: EventLevel,
    val time: String? = null
)

data class Config(
    val host: String,
    val port: Int = DEFAULT_PORT,
    val packageName: String,
    val reconnectOnDisconnect: Boolean = true,
    val reconnectDelayMs: Long = 3_000L,
    val sendHistoryOnConnect: Boolean = true,
)

private fun Event.toJson() = Json.encodeToString(this)

private fun String.toEventLevel(): EventLevel = when (this) {
    "D" -> EventLevel.DEBUG
    "I" -> EventLevel.INFO
    "E" -> EventLevel.ERROR
    "V" -> EventLevel.VERBOSE
    "W" -> EventLevel.WARNING
    else -> EventLevel.DEBUG
}
