package io.github.asvid.remoteloggerapp

import android.app.Application
import android.util.Log
import io.github.asvid.remotelogger.Config
import io.github.asvid.remotelogger.RemoteLogger
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.asCoroutineDispatcher
import java.util.concurrent.Executors

class App : Application() {

    override fun onCreate() {
        super.onCreate()
        Log.d("APP", "onCreate")
        RemoteLogger().initialize(
            Config(
                "192.168.1.87",
                1234,
                applicationContext.packageName
            )
        )
    }
}