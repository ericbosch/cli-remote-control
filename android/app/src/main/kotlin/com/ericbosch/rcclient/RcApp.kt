package com.ericbosch.rcclient

import android.app.Application

class RcApp : Application() {
    override fun onCreate() {
        super.onCreate()
        Preferences.init(this)
    }
}
