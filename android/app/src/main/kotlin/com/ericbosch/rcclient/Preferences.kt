package com.ericbosch.rcclient

import android.content.Context
import android.content.SharedPreferences

object Preferences {
    private const val PREFS_NAME = "rc_prefs"
    private const val KEY_BASE_URL = "base_url"
    private const val KEY_TOKEN = "token"

    private var appContext: Context? = null

    fun init(context: Context) {
        appContext = context.applicationContext
    }

    private fun prefs(): SharedPreferences {
        val ctx = appContext ?: throw IllegalStateException("Preferences not initialized")
        return ctx.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    }

    fun getBaseUrl(): String = prefs().getString(KEY_BASE_URL, "http://10.0.2.2:8787") ?: "http://10.0.2.2:8787"

    fun setBaseUrl(url: String) {
        prefs().edit().putString(KEY_BASE_URL, url).apply()
    }

    fun getToken(): String = prefs().getString(KEY_TOKEN, "") ?: ""

    fun setToken(token: String) {
        prefs().edit().putString(KEY_TOKEN, token).apply()
    }
}
