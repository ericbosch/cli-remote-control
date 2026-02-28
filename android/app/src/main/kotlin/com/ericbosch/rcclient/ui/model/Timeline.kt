package com.ericbosch.rcclient.ui.model

data class TimelineState(
    val connection: ConnectionState = ConnectionState.Idle,
    val sessionId: String = "",
    val title: String = "",
    val engine: String = "",
    val sessionState: String = "",
    val lastSeq: Long = 0,
    val rawMode: Boolean = false,
    val items: List<TimelineItem> = emptyList(),
)

enum class ConnectionState { Idle, Connecting, Connected, Reconnecting, Closed }

sealed interface TimelineItem {
    val tsMs: Long
    val seq: Long

    data class Status(override val tsMs: Long, override val seq: Long, val text: String) : TimelineItem
    data class Error(override val tsMs: Long, override val seq: Long, val text: String) : TimelineItem
    data class User(override val tsMs: Long, override val seq: Long, val text: String) : TimelineItem
    data class Assistant(override val tsMs: Long, override val seq: Long, val text: String) : TimelineItem
    data class Thinking(override val tsMs: Long, override val seq: Long, val delta: String, val done: Boolean) : TimelineItem
    data class Tool(override val tsMs: Long, override val seq: Long, val kind: String, val json: String) : TimelineItem
}

