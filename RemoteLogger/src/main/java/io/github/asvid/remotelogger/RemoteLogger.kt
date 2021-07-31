package io.github.asvid.remotelogger

import android.util.Log
import io.ktor.client.*
import io.ktor.client.features.websocket.*
import io.ktor.http.cio.websocket.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.asCoroutineDispatcher
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.io.BufferedReader
import java.io.InputStreamReader
import java.util.*
import java.util.concurrent.Executors
import kotlin.system.exitProcess

class RemoteLogger {
    private lateinit var config: Config
    private val client = HttpClient {
        install(WebSockets)
    }
    var session: DefaultClientWebSocketSession? = null

    private var events: Channel<Event>? = null

    fun initialize(config: Config) {
        this.config = config
        events = Channel<Event>()
        connect()
    }

    // todo: some reconect policy? WS pinging?
    fun connect() {
        config.coroutineScope.launch {
            readLogcatStream()
            client.webSocket(host = config.ip, port = config.port) {
                session = this
                startSendingLogsFromQueue()
            }
        }
        setAppCrashListener()
    }

    private fun setAppCrashListener() {
        Thread.setDefaultUncaughtExceptionHandler { paramThread, paramThrowable ->
            config.coroutineScope.launch {
                Log.e("AndroidRuntime", "--->uncaughtException:$paramThread<---", paramThrowable);
                val event = Event(
                    "CRASH",
                    paramThrowable.stackTraceToString(),
                    EventType.ERROR,
                    Date().toString()
                )
                sentEventViaWs(event)
                close()
                exitProcess(1)
            }
        }
    }

    fun close() {
        client.close()
    }

    private suspend fun logEvent(event: Event) {
        events?.send(event)
    }

    private suspend fun startSendingLogsFromQueue() {
        events?.receiveAsFlow()?.collect {
            if (client.isActive)
                sentEventViaWs(it)
            else {
                close()
            }
        }
    }

    private suspend fun sentEventViaWs(it: Event) {
        session?.send(it.toJson())
    }

    private fun readLogcatStream() {
        CoroutineScope(
            Executors.newSingleThreadExecutor().asCoroutineDispatcher()
        ).launch {
            cleanLogcat()

            val loggingProcess = Runtime.getRuntime()
                .exec("logcat ${config.packageName} -v time")

            val inputstr = InputStreamReader(loggingProcess.inputStream)
            val buff = BufferedReader(inputstr)
            while (true) {
                val line = buff.readLine()
                kotlin.runCatching {
                    val logcatRegex = """(.*) ([IDWVE])(\/.+)+\ *(\(\d*\)): (.*)""".toRegex()
                    val matches = logcatRegex.findAll(line).first()
                    val time = matches.groupValues[1]
                    val level = matches.groupValues[2]
                    val tag = matches.groupValues[3].drop(1)
                    val message = matches.groupValues[5]
                    logEvent(Event(tag, message, level.toEventType(), time))
                }.onFailure {
                    logEvent(Event("LOGGER", line, EventType.DEBUG, Date().toString()))
                }
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

enum class EventType { INFO, ERROR, DEBUG, VERBOSE, WARNING }

@Serializable
data class Event(
    val tag: String,
    val message: String,
    val type: EventType,
    val time: String? = null
)

data class Config(
    val ip: String,
    val port: Int,
    val coroutineScope: CoroutineScope,
    val packageName: String
)

private fun Event.toJson() = Json.encodeToString(this)
private fun String.toEventType(): EventType = when (this) {
    "D" -> EventType.DEBUG
    "I" -> EventType.INFO
    "E" -> EventType.ERROR
    "V" -> EventType.VERBOSE
    "W" -> EventType.WARNING
    else -> EventType.DEBUG
}