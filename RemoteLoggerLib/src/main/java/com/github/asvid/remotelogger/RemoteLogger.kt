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
import java.security.MessageDigest
import java.security.cert.CertificateException
import java.security.cert.X509Certificate
import java.util.*
import java.util.concurrent.TimeUnit
import javax.net.ssl.SSLContext
import javax.net.ssl.X509TrustManager
import kotlin.math.min
import kotlin.system.exitProcess

const val DEFAULT_PORT = 3000

class RemoteLogger {
    private lateinit var config: Config
    private lateinit var client: OkHttpClient
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private var webSocket: WebSocket? = null
    private var events: Channel<Event>? = null

    fun initialize(config: Config) {
        this.config = config
        this.client = buildClient(config)
        events = Channel(UNLIMITED)
        scope.launch { readLogcatStream() }
        scope.launch { connectWithBackoff() }
        setAppCrashListener()
    }

    private fun buildClient(config: Config): OkHttpClient {
        val builder = OkHttpClient.Builder()
            .pingInterval(20, TimeUnit.SECONDS)

        if (config.certPin != null) {
            val trustManager = PinnedCertTrustManager(config.certPin)
            val sslContext = SSLContext.getInstance("TLS")
            sslContext.init(null, arrayOf(trustManager), null)
            builder
                .sslSocketFactory(sslContext.socketFactory, trustManager)
                // Hostname verification is skipped because we connect to an IP address,
                // not a hostname. The cert is still validated against certPin.
                .hostnameVerifier { _, _ -> true }
        }

        return builder.build()
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

            val scheme = if (config.certPin != null) "wss" else "ws"

            suspendCancellableCoroutine<Boolean> { cont ->
                val request = Request.Builder()
                    .url("$scheme://${config.host}:${config.port}")
                    .build()

                val ws = client.newWebSocket(request, object : WebSocketListener() {
                    override fun onOpen(ws: WebSocket, response: Response) {
                        webSocket = ws
                        delayMs = config.reconnectDelayMs
                        Log.i("RemoteLogger", "Connected to ${config.host}:${config.port} ($scheme)")

                        if (!config.sendHistoryOnConnect) {
                            events?.tryReceive()
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

/**
 * Trusts only the certificate whose SubjectPublicKeyInfo SHA-256 hash matches [expectedPin].
 * The pin format is "sha256/base64==" — copy it from the server startup output.
 */
private class PinnedCertTrustManager(private val expectedPin: String) : X509TrustManager {

    override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String) {
        val cert = chain[0]
        // SubjectPublicKeyInfo is what publicKey.encoded returns for X.509 certs —
        // identical to what the Go server hashes with RawSubjectPublicKeyInfo.
        val spki = cert.publicKey.encoded
        val hash = MessageDigest.getInstance("SHA-256").digest(spki)
        val pin = "sha256/" + android.util.Base64.encodeToString(hash, android.util.Base64.NO_WRAP)

        if (pin != expectedPin) {
            throw CertificateException(
                "Certificate pin mismatch.\n  expected: $expectedPin\n  got:      $pin"
            )
        }
    }

    override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String) = Unit
    override fun getAcceptedIssuers(): Array<X509Certificate> = emptyArray()
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
    /** SHA-256 pin of the server certificate's public key — printed at server startup.
     *  Format: "sha256/base64==". When set, the connection uses WSS (encrypted).
     *  When null, plain WS is used (backward compatible, less secure). */
    val certPin: String? = null,
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
