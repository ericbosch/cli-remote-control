package com.ericbosch.rcclient

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

@Composable
fun SessionsScreen(
    onOpenSettings: () -> Unit,
    onAttach: (String) -> Unit
) {
    var sessions by mutableStateOf<List<Api.SessionInfo>>(emptyList())
    var loading by mutableStateOf(true)
    var error by mutableStateOf<String?>(null)
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
    }

    LaunchedEffect(Unit) {
        load()
        try {
            sessions = Api.listSessions()
        } catch (e: Exception) {
            error = e.message ?: "Failed to load"
        } finally {
            loading = false
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Sessions") },
                actions = {
                    IconButton(onClick = onOpenSettings) {
                        Text("⚙", modifier = Modifier.padding(8.dp))
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .padding(16.dp)
                .fillMaxSize()
        ) {
            if (error != null) {
                Text(error!!, color = androidx.compose.ui.graphics.Color.Red)
                Spacer(modifier = Modifier.height(8.dp))
            }
            Button(
                onClick = {
                    load()
                    scope.launch {
                        try {
                            val s = Api.createCursorSession()
                            sessions = Api.listSessions()
                            onAttach(s.id)
                        } catch (e: Exception) {
                            error = e.message
                        } finally {
                            loading = false
                        }
                    }
                },
                modifier = Modifier.fillMaxWidth()
            ) { Text("New Cursor session") }
            Spacer(modifier = Modifier.height(16.dp))
            if (loading && sessions.isEmpty()) {
                Column(
                    modifier = Modifier.fillMaxSize(),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    CircularProgressIndicator()
                }
            } else {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    items(sessions) { s ->
                        SessionCard(
                            session = s,
                            onAttach = { onAttach(s.id) },
                            onTerminate = {
                                scope.launch {
                                    try {
                                        Api.terminateSession(s.id)
                                        sessions = Api.listSessions()
                                    } catch (_: Exception) {}
                                }
                            }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SessionCard(
    session: Api.SessionInfo,
    onAttach: () -> Unit,
    onTerminate: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onAttach() },
        colors = CardDefaults.cardColors()
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(session.name.ifEmpty { session.id })
                Text(session.engine + " · " + session.state, style = androidx.compose.material3.MaterialTheme.typography.bodySmall)
            }
            Button(onClick = onAttach) { Text("Attach") }
            if (session.state != "exited") {
                Button(onClick = onTerminate, colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF5a2d2d))) {
                    Text("Terminate")
                }
            }
        }
    }
}
