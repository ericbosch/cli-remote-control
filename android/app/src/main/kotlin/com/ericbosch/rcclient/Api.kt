package com.ericbosch.rcclient

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.TimeUnit
import kotlin.text.Charsets

object Api {
    private val client = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    data class SessionInfo(
        val id: String,
        val name: String,
        val engine: String,
        val state: String,
        val exitCode: Int?,
        val created: String
    )

    private fun authHeader(): String {
        val token = Preferences.getToken()
        return if (token.startsWith("Bearer ")) token else "Bearer $token"
    }

    private fun baseUrl(): String = Preferences.getBaseUrl()

    suspend fun listSessions(): List<SessionInfo> = withContext(Dispatchers.IO) {
        val req = Request.Builder()
            .url("${baseUrl()}/api/sessions")
            .addHeader("Authorization", authHeader())
            .get()
            .build()
        val resp = client.newCall(req).execute()
        if (!resp.isSuccessful) throw ApiException(resp.code, resp.message)
        val body = resp.body?.string() ?: "[]"
        parseSessionList(body)
    }

    suspend fun createSession(engine: String = "shell", name: String? = null): SessionInfo = withContext(Dispatchers.IO) {
        val json = JSONObject().apply {
            put("engine", engine)
            name?.let { put("name", it) }
            put("args", JSONObject())
        }
        val req = Request.Builder()
            .url("${baseUrl()}/api/sessions")
            .addHeader("Authorization", authHeader())
            .addHeader("Content-Type", "application/json")
            .post(json.toString().toRequestBody("application/json".toMediaType()))
            .build()
        val resp = client.newCall(req).execute()
        if (!resp.isSuccessful) throw ApiException(resp.code, resp.message)
        val body = resp.body?.string() ?: throw ApiException(-1, "empty body")
        parseSession(body)
    }

    suspend fun terminateSession(id: String): Unit = withContext(Dispatchers.IO) {
        val req = Request.Builder()
            .url("${baseUrl()}/api/sessions/$id/terminate")
            .addHeader("Authorization", authHeader())
            .post("".toRequestBody(null))
            .build()
        val resp = client.newCall(req).execute()
        if (!resp.isSuccessful && resp.code != 404) throw ApiException(resp.code, resp.message)
    }

    fun wsUrl(sessionId: String): String {
        val base = baseUrl().replace("http://", "ws://").replace("https://", "wss://")
        val token = java.net.URLEncoder.encode(Preferences.getToken(), Charsets.UTF_8.name())
        return "$base/ws/sessions/$sessionId?token=$token"
    }

    private fun parseSessionList(json: String): List<SessionInfo> {
        val arr = JSONArray(json)
        return (0 until arr.length()).map { parseSession(arr.getJSONObject(it).toString()) }
    }

    private fun parseSession(json: String): SessionInfo {
        val o = JSONObject(json)
        return SessionInfo(
            id = o.optString("id", ""),
            name = o.optString("name", ""),
            engine = o.optString("engine", "shell"),
            state = o.optString("state", ""),
            exitCode = if (o.has("exit_code")) o.optInt("exit_code", 0) else null,
            created = o.optString("created", "")
        )
    }
}

class ApiException(val code: Int, message: String) : Exception("HTTP $code: $message")
