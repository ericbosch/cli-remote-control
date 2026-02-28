package com.ericbosch.rcclient

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

object Preferences {
    private const val PREFS_NAME = "rc_prefs"
    private const val KEY_BASE_URL = "base_url"
    private const val KEY_TOKEN = "token"
    private const val KEY_CONFIG_OK = "config_ok"

    private var appContext: Context? = null

    fun init(context: Context) {
        appContext = context.applicationContext
    }

    private fun prefs(): SharedPreferences {
        val ctx = appContext ?: throw IllegalStateException("Preferences not initialized")
        val masterKey = MasterKey.Builder(ctx).setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build()
        return EncryptedSharedPreferences.create(
            ctx,
            PREFS_NAME,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }

    fun getBaseUrl(): String = prefs().getString(KEY_BASE_URL, "") ?: ""

    fun setBaseUrl(url: String) {
        prefs().edit().putString(KEY_BASE_URL, url).apply()
    }

    fun getToken(): String = prefs().getString(KEY_TOKEN, "") ?: ""

    fun setToken(token: String) {
        prefs().edit().putString(KEY_TOKEN, token).apply()
    }

    fun isConfigOk(): Boolean = prefs().getBoolean(KEY_CONFIG_OK, false)

    fun setConfigOk(ok: Boolean) {
        prefs().edit().putBoolean(KEY_CONFIG_OK, ok).apply()
    }

    fun clearToken() {
        prefs().edit().remove(KEY_TOKEN).apply()
    }
}
