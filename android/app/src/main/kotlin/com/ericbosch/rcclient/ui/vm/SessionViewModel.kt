package com.ericbosch.rcclient.ui.vm

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ericbosch.rcclient.api.RcWsEventsStream
import com.ericbosch.rcclient.api.SessionEvent
import com.ericbosch.rcclient.api.RcApiClient
import com.ericbosch.rcclient.ui.model.ConnectionState
import com.ericbosch.rcclient.ui.model.TimelineItem
import com.ericbosch.rcclient.ui.model.TimelineState
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import okhttp3.OkHttpClient

class SessionViewModel(
    http: OkHttpClient,
    api: RcApiClient,
    private val json: Json,
) : ViewModel() {
    private val stream = RcWsEventsStream(viewModelScope, http, api, json)
    private val _state = MutableStateFlow(TimelineState())
    val state: StateFlow<TimelineState> = _state.asStateFlow()

    private var collectJob: Job? = null

    fun attach(sessionId: String, title: String, engine: String, sessionState: String, lastSeq: Long) {
        _state.value = _state.value.copy(
            sessionId = sessionId,
            title = title,
            engine = engine,
            sessionState = sessionState,
            lastSeq = lastSeq,
            items = emptyList(),
            connection = ConnectionState.Connecting,
            rawMode = false,
        )

        collectJob?.cancel()
        collectJob = viewModelScope.launch {
            stream.state.collect { s ->
                val c = when (s) {
                    is RcWsEventsStream.ConnState.Connecting -> ConnectionState.Connecting
                    is RcWsEventsStream.ConnState.Connected -> ConnectionState.Connected
                    is RcWsEventsStream.ConnState.Reconnecting -> ConnectionState.Reconnecting
                    is RcWsEventsStream.ConnState.Closed -> ConnectionState.Closed
                }
                _state.value = _state.value.copy(connection = c)
            }
        }

        viewModelScope.launch {
            stream.events.collect { ev ->
                reduceEvent(ev)
            }
        }

        stream.start(sessionId, lastSeq)
    }

    fun detach() {
        stream.stop()
        collectJob?.cancel()
        collectJob = null
        _state.value = TimelineState()
    }

    fun setRawMode(enabled: Boolean) {
        _state.value = _state.value.copy(rawMode = enabled)
    }

    fun sendLine(text: String) {
        val t = text.replace("\r\n", "\n")
        if (t.isBlank()) return
        val needsNewline = !_state.value.rawMode && _state.value.engine != "codex"
        stream.sendInput(if (needsNewline && !t.endsWith("\n")) t + "\n" else t)
        // Add a local user block (line-based) without per-keystroke spam.
        appendItem(TimelineItem.User(System.currentTimeMillis(), _state.value.lastSeq, t.trimEnd()))
    }

    fun sendCtrlC() {
        stream.sendInput("\u0003")
    }

    private fun reduceEvent(ev: SessionEvent) {
        val nextSeq = maxOf(_state.value.lastSeq, ev.seq)
        _state.value = _state.value.copy(lastSeq = nextSeq)

        when (ev.kind) {
            "assistant" -> {
                val data = payloadString(ev, "data")
                if (data.isNotEmpty()) appendOrMergeAssistant(ev.tsMs, ev.seq, data)
            }
            "status" -> {
                val st = payloadString(ev, "state")
                val code = payloadInt(ev, "exit_code")
                val txt = if (code != null) "$st (exit $code)" else st
                if (txt.isNotEmpty()) appendItem(TimelineItem.Status(ev.tsMs, ev.seq, txt))
            }
            "error" -> {
                val msg = payloadString(ev, "message").ifEmpty { payloadString(ev, "data") }
                if (msg.isNotEmpty()) appendItem(TimelineItem.Error(ev.tsMs, ev.seq, msg))
            }
            "thinking_delta" -> {
                val d = payloadString(ev, "delta")
                if (d.isNotEmpty()) appendItem(TimelineItem.Thinking(ev.tsMs, ev.seq, d, done = false))
            }
            "thinking_done" -> appendItem(TimelineItem.Thinking(ev.tsMs, ev.seq, "", done = true))
            "tool_call" -> appendTool(ev, "tool_call")
            "tool_output" -> appendTool(ev, "tool_output")
            "user" -> {
                // Ignore remote user echo; we add a local line block when sending.
            }
        }
    }

    private fun appendTool(ev: SessionEvent, kind: String) {
        val p = ev.payload
        val pretty = p?.toString() ?: ""
        appendItem(TimelineItem.Tool(ev.tsMs, ev.seq, kind, pretty))
    }

    private fun appendOrMergeAssistant(tsMs: Long, seq: Long, text: String) {
        val items = _state.value.items
        val last = items.lastOrNull()
        if (last is TimelineItem.Assistant && (seq - last.seq) <= 2) {
            val merged = last.copy(text = last.text + text)
            _state.value = _state.value.copy(items = items.dropLast(1) + merged)
        } else {
            appendItem(TimelineItem.Assistant(tsMs, seq, text))
        }
    }

    private fun appendItem(item: TimelineItem) {
        val items = _state.value.items
        val capped = (items + item).takeLast(2000)
        _state.value = _state.value.copy(items = capped)
    }

    private fun payloadString(ev: SessionEvent, key: String): String {
        val obj = ev.payload as? JsonObject ?: return ""
        val v = obj[key] as? JsonPrimitive ?: return ""
        val c = v.content
        return if (c == "null") "" else c
    }

    private fun payloadInt(ev: SessionEvent, key: String): Int? {
        val obj = ev.payload as? JsonObject ?: return null
        val v = obj[key] as? JsonPrimitive ?: return null
        return v.content.toIntOrNull()
    }
}
