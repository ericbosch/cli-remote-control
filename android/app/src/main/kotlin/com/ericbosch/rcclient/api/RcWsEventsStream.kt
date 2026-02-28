package com.ericbosch.rcclient.api

import com.ericbosch.rcclient.Preferences
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

class RcWsEventsStream(
    private val scope: CoroutineScope,
    private val http: OkHttpClient,
    private val api: RcApiClient,
    private val json: Json,
) {
    sealed class ConnState {
        data object Connecting : ConnState()
        data object Connected : ConnState()
        data class Reconnecting(val attempt: Int) : ConnState()
        data class Closed(val reason: String) : ConnState()
    }

    val events = MutableSharedFlow<SessionEvent>(extraBufferCapacity = 512)
    private val _state = MutableStateFlow<ConnState>(ConnState.Closed("init"))
    val state: StateFlow<ConnState> = _state

    private var ws: WebSocket? = null
    private var lastSeq: Long = 0
    private var runJob: Job? = null

    fun start(sessionId: String, initialLastSeq: Long) {
        stop()
        lastSeq = initialLastSeq
        runJob = scope.launch { runLoop(sessionId) }
    }

    fun stop() {
        runJob?.cancel()
        runJob = null
        ws?.close(1000, "stop")
        ws = null
        _state.value = ConnState.Closed("stopped")
    }

    fun sendInput(text: String) {
        val msg = WsClientMessage(type = "input", data = text)
        val payload = json.encodeToString(WsClientMessage.serializer(), msg)
        ws?.send(payload)
    }

    private suspend fun runLoop(sessionId: String) {
        var attempt = 0
        while (scope.isActive) {
            _state.value = if (attempt == 0) ConnState.Connecting else ConnState.Reconnecting(attempt)

            val ticketRes = api.issueWsTicket()
            val ticket = ticketRes.getOrNull()
            if (ticket == null) {
                val e = ticketRes.exceptionOrNull()
                _state.value = ConnState.Closed("ticket: ${e?.message ?: "failed"}")
                delay(backoff(attempt))
                attempt++
                continue
            }

            val wsUrl = buildWsUrl(sessionId, ticket, lastSeq)
            if (wsUrl == null) {
                _state.value = ConnState.Closed("invalid base url")
                delay(backoff(attempt))
                attempt++
                continue
            }

            val req = Request.Builder().url(wsUrl).build()
            val done = MutableStateFlow(false)
            ws = http.newWebSocket(req, object : WebSocketListener() {
                override fun onOpen(webSocket: WebSocket, response: Response) {
                    _state.value = ConnState.Connected
                    attempt = 0
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    runCatching { json.decodeFromString(SessionEvent.serializer(), text) }
                        .onSuccess { ev ->
                            if (ev.seq > lastSeq) lastSeq = ev.seq
                            events.tryEmit(ev)
                        }
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    _state.value = ConnState.Closed(t.message ?: "ws failure")
                    done.value = true
                }

                override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                    webSocket.close(code, reason)
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    _state.value = ConnState.Closed("closed $code")
                    done.value = true
                }
            })

            // Wait until closed/failure then loop.
            while (scope.isActive && !done.value) delay(100)
            ws = null
            delay(backoff(attempt))
            attempt++
        }
    }

    private fun backoff(attempt: Int): Long {
        val ms = when {
            attempt <= 0 -> 200L
            attempt == 1 -> 500L
            attempt == 2 -> 1000L
            attempt == 3 -> 2000L
            else -> 5000L
        }
        return ms
    }

    private fun buildWsUrl(sessionId: String, ticket: String, lastSeq: Long): String? {
        val base = Preferences.getBaseUrl().trim().removeSuffix("/")
        val httpUrl = base.toHttpUrlOrNull() ?: return null
        val scheme = if (httpUrl.scheme == "https") "wss" else "ws"
        val b = httpUrl.newBuilder()
            .scheme(scheme)
            .encodedPath("/ws/events/$sessionId")

        if (lastSeq > 0) {
            b.addQueryParameter("from_seq", (lastSeq + 1).toString())
        } else {
            b.addQueryParameter("last_n", "256")
        }
        b.addQueryParameter("ticket", ticket)
        return b.build().toString()
    }
}
