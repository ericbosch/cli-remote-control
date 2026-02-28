package com.ericbosch.rcclient.ui.screens

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Scaffold
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.Text
import androidx.compose.material3.IconButton
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Settings
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.material3.windowsizeclass.WindowWidthSizeClass
import androidx.compose.material3.windowsizeclass.WindowSizeClass
import com.ericbosch.rcclient.di.LocalDeps
import com.ericbosch.rcclient.ui.vm.SessionViewModel
import com.ericbosch.rcclient.ui.vm.SessionViewModelFactory
import com.ericbosch.rcclient.ui.vm.SessionsViewModel
import com.ericbosch.rcclient.ui.vm.SessionsViewModelFactory

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    windowSize: WindowSizeClass,
    onOpenSettings: () -> Unit,
) {
    val deps = LocalDeps.current
    val sessionsVm: SessionsViewModel = viewModel(factory = SessionsViewModelFactory(deps.api))
    val sessionVm: SessionViewModel = viewModel(factory = SessionViewModelFactory(deps.http, deps.api, deps.json))

    var selectedSessionId by remember { mutableStateOf<String?>(null) }

    val dualPane = windowSize.widthSizeClass >= WindowWidthSizeClass.Medium

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Remote Control") },
                actions = {
                    IconButton(onClick = onOpenSettings) {
                        androidx.compose.material3.Icon(Icons.Filled.Settings, contentDescription = "Settings")
                    }
                }
            )
        }
    ) { padding ->
        if (dualPane) {
            Row(modifier = Modifier.fillMaxSize()) {
                SessionsListPane(
                    modifier = Modifier.weight(0.42f),
                    padding = padding,
                    sessionsVm = sessionsVm,
                    selectedId = selectedSessionId,
                    onSelect = { s ->
                        selectedSessionId = s.id
                        sessionVm.attach(
                            sessionId = s.id,
                            title = if (s.name.isNotBlank()) s.name else s.id,
                            engine = s.engine,
                            sessionState = s.state,
                            lastSeq = s.lastSeq ?: 0,
                        )
                    },
                )
                SessionDetailPane(
                    modifier = Modifier.weight(0.58f),
                    padding = padding,
                    sessionVm = sessionVm,
                    onBack = { /* no-op */ },
                )
            }
        } else {
            Box(modifier = Modifier.fillMaxSize()) {
                if (selectedSessionId == null) {
                    SessionsListPane(
                        modifier = Modifier.fillMaxSize(),
                        padding = padding,
                        sessionsVm = sessionsVm,
                        selectedId = null,
                        onSelect = { s ->
                            selectedSessionId = s.id
                            sessionVm.attach(
                                sessionId = s.id,
                                title = if (s.name.isNotBlank()) s.name else s.id,
                                engine = s.engine,
                                sessionState = s.state,
                                lastSeq = s.lastSeq ?: 0,
                            )
                        },
                    )
                } else {
                    SessionDetailPane(
                        modifier = Modifier.fillMaxSize(),
                        padding = padding,
                        sessionVm = sessionVm,
                        onBack = {
                            selectedSessionId = null
                            sessionVm.detach()
                        },
                    )
                }
            }
        }
    }
}
