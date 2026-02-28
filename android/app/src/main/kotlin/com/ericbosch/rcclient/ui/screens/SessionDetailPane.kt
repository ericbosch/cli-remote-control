package com.ericbosch.rcclient.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.Switch
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.collectAsState
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.ericbosch.rcclient.ui.model.ConnectionState
import com.ericbosch.rcclient.ui.model.TimelineItem
import com.ericbosch.rcclient.ui.vm.SessionViewModel
import androidx.compose.runtime.rememberCoroutineScope
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SessionDetailPane(
    modifier: Modifier,
    padding: PaddingValues,
    sessionVm: SessionViewModel,
    onBack: () -> Unit,
) {
    val state by sessionVm.state.collectAsState()
    val listState = rememberLazyListState()
    var input by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    val atBottom by remember {
        derivedStateOf {
            val info = listState.layoutInfo
            val lastVisible = info.visibleItemsInfo.lastOrNull()?.index ?: 0
            lastVisible >= (info.totalItemsCount - 2)
        }
    }

    LaunchedEffect(state.items.size) {
        if (atBottom && state.items.isNotEmpty()) {
            listState.animateScrollToItem(state.items.size - 1)
        }
    }

    Column(modifier = modifier.fillMaxHeight().padding(padding).padding(12.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(bottom = 8.dp)) {
            IconButton(onClick = onBack) { Icon(Icons.Filled.ArrowBack, contentDescription = "Back") }
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    state.title.ifBlank { "No session selected" },
                    style = MaterialTheme.typography.titleMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    "${state.engine} · ${state.sessionState} · ${state.connection.name.lowercase()}",
                    style = MaterialTheme.typography.bodySmall
                )
            }
        }

        Surface(
            tonalElevation = 2.dp,
            modifier = Modifier.weight(1f).fillMaxSize()
        ) {
            Box(modifier = Modifier.fillMaxSize()) {
                TimelineList(state.items, listState)
                if (!atBottom && state.items.isNotEmpty()) {
                    IconButton(
                        onClick = {
                            scope.launch { listState.animateScrollToItem(state.items.size - 1) }
                        },
                        modifier = Modifier.align(Alignment.BottomEnd).padding(8.dp)
                    ) {
                        Icon(Icons.Filled.KeyboardArrowDown, contentDescription = "Jump to bottom")
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(10.dp))

        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedTextField(
                value = input,
                onValueChange = { input = it },
                modifier = Modifier.weight(1f),
                label = { Text("Send input") },
                singleLine = true,
            )
            Button(
                onClick = {
                    val v = input
                    input = ""
                    sessionVm.sendLine(v)
                },
                enabled = state.sessionId.isNotBlank() && state.connection != ConnectionState.Closed
            ) { Text("Send") }
        }

        Row(
            modifier = Modifier.padding(top = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            OutlinedButton(onClick = { sessionVm.sendCtrlC() }, enabled = state.sessionId.isNotBlank()) { Text("Ctrl+C") }
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Raw", style = MaterialTheme.typography.bodySmall)
                Spacer(Modifier.width(6.dp))
                Switch(checked = state.rawMode, onCheckedChange = { sessionVm.setRawMode(it) }, enabled = state.sessionId.isNotBlank())
            }
        }
    }
}

@Composable
private fun TimelineList(items: List<TimelineItem>, listState: LazyListState) {
    LazyColumn(
        state = listState,
        modifier = Modifier.fillMaxSize().padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp)
    ) {
        items(items.size) { i ->
            when (val it = items[i]) {
                is TimelineItem.Status -> Block("Status", it.text)
                is TimelineItem.Error -> Block("Error", it.text, isError = true)
                is TimelineItem.User -> Block("You", it.text)
                is TimelineItem.Assistant -> Block("Output", it.text, monospace = true)
                is TimelineItem.Thinking -> {
                    if (it.done) Block("Thinking", "(done)", muted = true) else Block("Thinking", it.delta, muted = true)
                }
                is TimelineItem.Tool -> Block(it.kind, it.json, monospace = true, muted = true)
            }
        }
    }
}

@Composable
private fun Block(title: String, body: String, monospace: Boolean = false, muted: Boolean = false, isError: Boolean = false) {
    val bg = when {
        isError -> MaterialTheme.colorScheme.errorContainer
        muted -> MaterialTheme.colorScheme.surfaceVariant
        else -> MaterialTheme.colorScheme.surface
    }
    val fg = when {
        isError -> MaterialTheme.colorScheme.onErrorContainer
        muted -> MaterialTheme.colorScheme.onSurfaceVariant
        else -> MaterialTheme.colorScheme.onSurface
    }

    Surface(color = bg, tonalElevation = 1.dp, shape = MaterialTheme.shapes.medium) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(title, style = MaterialTheme.typography.labelMedium, color = fg)
            Spacer(Modifier.height(6.dp))
            Text(
                body,
                style = MaterialTheme.typography.bodyMedium,
                color = fg,
                fontFamily = if (monospace) FontFamily.Monospace else null,
            )
        }
    }
}
