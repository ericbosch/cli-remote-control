package com.ericbosch.rcclient.ui.screens

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.clickable
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material3.Icon
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.LaunchedEffect
import com.ericbosch.rcclient.api.CreateSessionRequest
import com.ericbosch.rcclient.api.SessionInfo
import com.ericbosch.rcclient.di.LocalDeps
import com.ericbosch.rcclient.ui.vm.SessionsViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SessionsListPane(
    modifier: Modifier,
    padding: PaddingValues,
    sessionsVm: SessionsViewModel,
    selectedId: String?,
    onSelect: (SessionInfo) -> Unit,
) {
    val state by sessionsVm.state.collectAsState()
    var createOpen by remember { mutableStateOf(false) }

    Column(modifier = modifier.fillMaxHeight().padding(padding).padding(12.dp)) {
        Text("Sessions", style = MaterialTheme.typography.titleMedium)
        if (state.error != null) {
            Text(state.error ?: "", color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(top = 8.dp))
        }
        OutlinedButton(
            onClick = { createOpen = true },
            modifier = Modifier.padding(top = 10.dp)
        ) { Text("New session") }

        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(top = 10.dp),
        ) {
            items(state.sessions, key = { it.id }) { s ->
                val isSelected = selectedId == s.id
                Card(
                    onClick = { onSelect(s) },
                    colors = CardDefaults.cardColors(
                        containerColor = if (isSelected) MaterialTheme.colorScheme.secondaryContainer else MaterialTheme.colorScheme.surface
                    ),
                    modifier = Modifier.padding(bottom = 8.dp),
                ) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Text(if (s.name.isNotBlank()) s.name else s.id, style = MaterialTheme.typography.titleSmall)
                        Text("${s.engine} · ${s.state}", style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
        }
    }

    if (createOpen) {
        CreateSessionDialog(
            onDismiss = { createOpen = false },
            onCreate = { engine, name, workspace, prompt ->
                sessionsVm.createSession(
                    CreateSessionRequest(engine = engine, name = name, workspacePath = workspace, prompt = prompt),
                    onCreated = { created ->
                        createOpen = false
                        onSelect(created)
                    }
                )
            }
        )
    }
}

@Composable
private fun CreateSessionDialog(
    onDismiss: () -> Unit,
    onCreate: (engine: String, name: String, workspace: String, prompt: String) -> Unit,
) {
    val deps = LocalDeps.current
    var engines by remember { mutableStateOf(listOf("shell")) }
    var engine by remember { mutableStateOf("shell") }
    var engineExpanded by remember { mutableStateOf(false) }
    var name by remember { mutableStateOf("") }
    var workspace by remember { mutableStateOf("") }
    var prompt by remember { mutableStateOf("") }

    LaunchedEffect(Unit) {
        deps.api.listEngines().onSuccess { list ->
            val cleaned = list.map { it.trim() }.filter { it.isNotBlank() }.distinct()
            engines = if (cleaned.isEmpty()) listOf("shell") else cleaned
            if (engine !in engines) engine = engines.first()
        }
    }

    androidx.compose.material3.AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("New session") },
        text = {
            Column {
                Box {
                    OutlinedTextField(
                        value = engine,
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("Engine") },
                        trailingIcon = { Icon(Icons.Default.ArrowDropDown, contentDescription = null) },
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { engineExpanded = true },
                        singleLine = true
                    )
                    DropdownMenu(
                        expanded = engineExpanded,
                        onDismissRequest = { engineExpanded = false }
                    ) {
                        engines.forEach { e ->
                            DropdownMenuItem(
                                text = { Text(e) },
                                onClick = {
                                    engine = e
                                    engineExpanded = false
                                }
                            )
                        }
                    }
                }
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name (optional)") }, singleLine = true)
                OutlinedTextField(value = workspace, onValueChange = { workspace = it }, label = { Text("Workspace (optional)") }, singleLine = true)
                OutlinedTextField(value = prompt, onValueChange = { prompt = it }, label = { Text("Prompt (optional)") })
            }
        },
        confirmButton = {
            Button(onClick = { onCreate(engine.trim().ifBlank { "shell" }, name.trim(), workspace.trim(), prompt) }) { Text("Create") }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Cancel") }
        }
    )
}
