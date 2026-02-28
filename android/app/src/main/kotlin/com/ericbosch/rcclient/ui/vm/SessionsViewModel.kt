package com.ericbosch.rcclient.ui.vm

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.ericbosch.rcclient.api.CreateSessionRequest
import com.ericbosch.rcclient.api.RcApiClient
import com.ericbosch.rcclient.api.SessionInfo
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

data class SessionsUiState(
    val loading: Boolean = false,
    val error: String? = null,
    val sessions: List<SessionInfo> = emptyList(),
)

class SessionsViewModel(
    private val api: RcApiClient,
) : ViewModel() {
    private val _state = MutableStateFlow(SessionsUiState(loading = true))
    val state: StateFlow<SessionsUiState> = _state.asStateFlow()

    init {
        refresh()
        viewModelScope.launch {
            while (true) {
                delay(4000)
                refresh()
            }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            _state.value = _state.value.copy(loading = true, error = null)
            val r = api.listSessions()
            _state.value = r.fold(
                onSuccess = { SessionsUiState(loading = false, sessions = it, error = null) },
                onFailure = { SessionsUiState(loading = false, sessions = _state.value.sessions, error = it.message ?: "Failed") },
            )
        }
    }

    fun createSession(req: CreateSessionRequest, onCreated: (SessionInfo) -> Unit) {
        viewModelScope.launch {
            _state.value = _state.value.copy(loading = true, error = null)
            api.createSession(req).fold(
                onSuccess = {
                    onCreated(it)
                    refresh()
                },
                onFailure = { _state.value = _state.value.copy(loading = false, error = it.message ?: "Create failed") },
            )
        }
    }
}

