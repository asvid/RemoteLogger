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
                "192.168.1.87",
                1234,
                applicationContext.packageName
            )
        )
    }
}