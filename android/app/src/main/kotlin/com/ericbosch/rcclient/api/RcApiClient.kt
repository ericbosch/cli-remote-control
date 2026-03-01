package com.ericbosch.rcclient.api

import com.ericbosch.rcclient.Preferences
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

class RcApiClient(
    private val http: OkHttpClient,
    private val json: Json,
) {
    private fun baseUrl(): String = Preferences.getBaseUrl().trim().removeSuffix("/")
    private fun token(): String = Preferences.getToken().trim()

    private fun requestBuilder(path: String): Request.Builder {
        val base = baseUrl()
        val url = (base + path).toHttpUrlOrNull() ?: throw IllegalStateException("Invalid base URL")
        val b = Request.Builder().url(url)
        val t = token()
        if (t.isNotEmpty()) b.header("Authorization", if (t.startsWith("Bearer ")) t else "Bearer $t")
        return b
    }

    suspend fun listSessions(): Result<List<SessionInfo>> = withContext(Dispatchers.IO) {
        runCatching {
            val req = requestBuilder("/api/sessions").get().build()
            httpCall(req) { body ->
                val list = json.decodeFromString(kotlinx.serialization.builtins.ListSerializer(SessionInfo.serializer()), body)
                Result.success(list)
            }
        }.getOrElse { Result.failure(it) }
    }

    suspend fun createSession(reqBody: CreateSessionRequest): Result<SessionInfo> = withContext(Dispatchers.IO) {
        runCatching {
            val media = "application/json".toMediaType()
            val payload = json.encodeToString(CreateSessionRequest.serializer(), reqBody).toRequestBody(media)
            val req = requestBuilder("/api/sessions").post(payload).build()
            httpCall(req) { body ->
                val info = json.decodeFromString(SessionInfo.serializer(), body)
                Result.success(info)
            }
        }.getOrElse { Result.failure(it) }
    }

    suspend fun issueWsTicket(): Result<String> = withContext(Dispatchers.IO) {
        runCatching {
            val req = requestBuilder("/api/ws-ticket").post("{}".toRequestBody("application/json".toMediaType())).build()
            httpCall(req) { body ->
                val obj = json.decodeFromString(WsTicketResponse.serializer(), body)
                val ticket = obj.ticket?.trim().orEmpty()
                if (ticket.isEmpty()) Result.failure(IllegalStateException("Missing ws ticket")) else Result.success(ticket)
            }
        }.getOrElse { Result.failure(it) }
    }

    suspend fun listEngines(): Result<List<String>> = withContext(Dispatchers.IO) {
        runCatching {
            val req = requestBuilder("/api/engines").get().build()
            httpCall(req) { body ->
                val list = json.decodeFromString(ListSerializer(String.serializer()), body)
                Result.success(list)
            }
        }.getOrElse { Result.failure(it) }
    }

    suspend fun healthz(): Result<Boolean> = withContext(Dispatchers.IO) {
        runCatching {
            val req = run {
                val base = baseUrl()
                val url = (base + "/healthz").toHttpUrlOrNull() ?: throw IllegalStateException("Invalid base URL")
                Request.Builder().url(url).get().build()
            }
            val res = http.newCall(req).execute()
            res.use { Result.success(it.isSuccessful) }
        }.getOrElse { Result.failure(it) }
    }

    private inline fun <T> httpCall(request: Request, decode: (String) -> Result<T>): Result<T> {
        val res: Response = try {
            http.newCall(request).execute()
        } catch (e: Exception) {
            return Result.failure(e)
        }
        res.use {
            val body = it.body?.string().orEmpty()
            if (it.isSuccessful) {
                return try {
                    decode(body)
                } catch (e: Exception) {
                    Result.failure(e)
                }
            }

            // Prefer structured API errors (no secrets expected).
            val parsed = runCatching { json.decodeFromString(ApiErrorEnvelope.serializer(), body) }.getOrNull()
            val e = parsed?.error
            if (e != null) {
                val msg = listOfNotNull(
                    e.message?.takeIf { s -> s.isNotBlank() },
                    e.hint?.takeIf { s -> s.isNotBlank() }?.let { "Hint: $it" },
                    e.requestId?.takeIf { s -> s.isNotBlank() }?.let { "request_id=$it" }
                ).joinToString(" · ")
                return Result.failure(IllegalStateException("${e.code ?: "error"} (${it.code}): $msg"))
            }
            return Result.failure(IllegalStateException("HTTP ${it.code}"))
        }
    }
}
