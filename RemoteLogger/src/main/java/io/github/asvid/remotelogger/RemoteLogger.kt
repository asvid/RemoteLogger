package io.github.asvid.remotelogger

import android.util.Log
import io.github.asvid.remotelogger.fifo.FifoQueue
import io.github.asvid.remotelogger.fifo.FifoQueueImpl
import io.ktor.client.*
import io.ktor.client.features.websocket.*
import io.ktor.http.cio.websocket.*
import kotlinx.coroutines.*
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.channels.consumeEach
import java.io.BufferedReader
import java.io.InputStreamReader
import java.lang.Exception
import java.util.concurrent.Executors

object RemoteLogger {
    private lateinit var config: Config
    private val client = HttpClient {
        install(WebSockets)
    }
    var session: DefaultClientWebSocketSession? = null

    val events = Channel<Event>()

    fun initialize(config: Config) {
        // start websocket at provided IP
        // no need to listen to it, maybe just ot get handshake
        // start sending messages
        this.config = config
        config.coroutineScope.launch {
            readLogcatStream()
            client.webSocket(host = config.ip, port = config.port) {
                session = this
                this.send("we have a connection from Android")
                readLogcatStream()
                startSendingLogsFromQueue()
            }
        }
    }

    private suspend fun logEvent(event: Event) {
        events.send(event)
    }

    private suspend fun startSendingLogsFromQueue() {
        events.consumeEach {
            session?.send(it.toString())
        }
    }

    private fun readLogcatStream() {
        CoroutineScope(
            Executors.newSingleThreadExecutor().asCoroutineDispatcher()).launch{

            val loggingProcess = Runtime.getRuntime().exec("logcat ${config.packageName} -v time")

            val inputstr = InputStreamReader(loggingProcess.inputStream)
            val buff = BufferedReader(inputstr)
            while(true){
                val line = buff.readLine()
                logEvent(Event("LOGCAT", line))
            }
        }
    }
}

// todo: create global exception handler that will capture last exception and send it before app crashes

data class Event(val tag: String, val message: String)

data class Config(
    val ip: String,
    val port: Int,
    val coroutineScope: CoroutineScope,
    val packageName: String
)