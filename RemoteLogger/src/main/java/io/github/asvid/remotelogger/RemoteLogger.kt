package io.github.asvid.remotelogger

import io.ktor.client.*
import io.ktor.client.features.websocket.*
import io.ktor.http.cio.websocket.*
import kotlinx.coroutines.*
import java.lang.Exception

object RemoteLogger {
    private lateinit var config: Config
    private val client = HttpClient {
        install(WebSockets)
    }
    var session: DefaultClientWebSocketSession? = null

    fun doStuff() = "hey mate!"

    fun initialize(config: Config) {
        // start websocket at provided IP
        // no need to listen to it, maybe just ot get handshake
        // start sending messages
        this.config = config
        config.coroutineScope.launch {
            client.webSocket(host = config.ip, port = config.port) {
                session = this
                this.send("we have a connection from Android")
            }
        }
    }

    fun logEvent(event: Event) {
        if (session == null) throw Exception("not initialized!")
        // todo: cache events untill its possible to send them, nothing can be lost
        config.coroutineScope.launch {
            session?.send(event.toString())
        }
    }
}

// todo: create global exception handler that will capture last exception and send it before app crashes

data class Event(val tag: String, val message: String)

data class Config(val ip: String, val port: Int, val coroutineScope: CoroutineScope)