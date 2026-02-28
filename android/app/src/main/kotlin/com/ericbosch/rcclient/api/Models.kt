package com.ericbosch.rcclient.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class SessionInfo(
    val id: String,
    val name: String = "",
    val engine: String = "",
    val state: String = "",
    @SerialName("exit_code") val exitCode: Int? = null,
    @SerialName("last_seq") val lastSeq: Long? = null,
    val created: String = "",
)

@Serializable
data class CreateSessionRequest(
    val engine: String,
    val name: String = "",
    val workspacePath: String = "",
    val prompt: String = "",
    val mode: String = "",
    val args: Map<String, JsonElement> = emptyMap(),
)

@Serializable
data class WsTicketResponse(val ticket: String? = null)

@Serializable
data class ApiErrorEnvelope(val error: ApiErrorPayload? = null)

@Serializable
data class ApiErrorPayload(
    val code: String? = null,
    val message: String? = null,
    val hint: String? = null,
    @SerialName("request_id") val requestId: String? = null,
)

@Serializable
data class SessionEvent(
    @SerialName("session_id") val sessionId: String,
    val engine: String,
    @SerialName("ts_ms") val tsMs: Long,
    val seq: Long,
    val kind: String,
    val payload: JsonElement? = null,
)

@Serializable
data class WsClientMessage(
    val type: String,
    val data: String = "",
    val cols: Int = 0,
    val rows: Int = 0,
    val ts: Long = 0,
)

