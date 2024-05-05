# RemoteLogger [![](https://jitpack.io/v/asvid/RemoteLogger.svg)](https://jitpack.io/#asvid/RemoteLogger)

Little tool that allows Android apps to send Logcat logs via websocket to server hosted on a developer machine - useful when the app has to be in release mode without developer options turned on (real live case...)

## How to run

There is legacy ExpressJS app in `/server`, and a rewrite in Go in `/serverGo`.

### Run server first

#### NodeJS

Go to `server` directory and run `npm install` to get all dependencies.
Then run `node index.js`. If you want to run at specific port, then use `WS_PORT=1111 node index.js`.
Server will open the browser with logs page, but it won't refresh automatically, you have to do it manually.

![terminal look](./readme/terminal.png)
This service creates a website on localhost that displays all logs. You have to refresh it manually.
![bui](./readme/bui.png)

#### Go

Simply run `./serverGo/remotelogger` from command line. There are few optional parameters available:

```
./remotelogger -h
______                          _            _
| ___ \                        | |          | |
| |_/ /  ___  _ __ ___    ___  | |_   ___   | |      ___    __ _   __ _   ___  _ __
|    /  / _ \| '_ ` _ \  / _ \ | __| / _ \  | |     / _ \  / _` | / _` | / _ \| '__|
| |\ \ |  __/| | | | | || (_) || |_ |  __/  | |____| (_) || (_| || (_| ||  __/| |
\_| \_| \___||_| |_| |_| \___/  \__| \___|  \_____/ \___/  \__, | \__, | \___||_|
                                                            __/ |  __/ |
                                                           |___/  |___/
Usage of ./remotelogger:
  -f string
    	Save logs to file with povided name. Log filters do not apply here.
  -ll string
    	Display only logs at provided level, available options: INFO, ERROR, DEBUG, VERBOSE, WARNING.
  -p int
    	Set port for running Remote Logger. (default 3000)
  -t string
    	Display only logs with provided tag.
```

This app doesn't provide browser user interface, logs are displayed in terminal when the server is running, or saved to file.

### Android

Add Jitpack to your repositories in root `build.gradle` or `settings.gradle`:

```groovy
allprojects {
    repositories {
        ...
        maven { url 'https://jitpack.io' }
    }
}
```

And library dependency in project:

```groovy
	dependencies {
	        implementation 'com.github.asvid:RemoteLogger:0.2.0'
	}
```

In Android app, init the library in the Application class `onCreate()` method, like this:

```kotlin
RemoteLogger().initialize(
    Config(
        "192.168.1.87", // your server IP, will be displayed in CMD after server is started
        1234, // optional port
        applicationContext.packageName // package name, required for proper LogCat filtering
    )
)
```

After `initialize()` is called, websocket tries to connect to server so it has to be started first.
After it connects it sends all the logs that were already gathered by LogCat, so nothing is lost.
Each application restart clears the logs.

When app crashes the exception is also logged remotely, and then app exits with code `1`.

You can still see the logs in Android Studio Logcat, the hijacking is transparent :)

# Licence

MIT - Check the details [here](./LICENSE)
