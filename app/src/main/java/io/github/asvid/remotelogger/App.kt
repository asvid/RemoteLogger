package io.github.asvid.remotelogger

import android.app.Application
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.asCoroutineDispatcher
import kotlinx.coroutines.currentCoroutineContext
import java.util.concurrent.Executors

class App : Application() {

    override fun onCreate() {
        super.onCreate()
        Log.d("APP", "onCreate")
        RemoteLogger.initialize(
            Config(
                "192.168.1.87",
                1234,
                CoroutineScope(Executors.newSingleThreadExecutor().asCoroutineDispatcher()),
                applicationContext.packageName
            )
        )
    }
}