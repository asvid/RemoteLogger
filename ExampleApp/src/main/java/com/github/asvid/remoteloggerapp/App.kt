package com.github.asvid.remoteloggerapp

import android.app.Application
import android.util.Log
import com.github.asvid.remotelogger.Config
import com.github.asvid.remotelogger.RemoteLogger

class App : Application() {

    override fun onCreate() {
        super.onCreate()
        RemoteLogger().initialize(
            Config(
                host = "192.168.100.4",
                port = 3000,
                packageName = applicationContext.packageName,
                reconnectOnDisconnect = true,
            )
        )
    }
}