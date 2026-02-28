package com.ericbosch.rcclient.ui.screens

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.text.KeyboardOptions
import com.ericbosch.rcclient.Preferences
import com.ericbosch.rcclient.di.LocalDeps
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SetupScreen(
    onContinue: () -> Unit,
) {
    val deps = LocalDeps.current
    val scope = rememberCoroutineScope()

    var baseUrl by rememberSaveable { mutableStateOf(Preferences.getBaseUrl()) }
    var token by rememberSaveable { mutableStateOf("") } // never prefill token
    var validating by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    val suggested = "https://krinekk-hp-elitebook-850-g5.tail353084.ts.net:8443"

    Scaffold(
        topBar = { TopAppBar(title = { Text("Connect") }) }
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .padding(16.dp)
                .fillMaxWidth()
        ) {
            Text("Base URL")
            OutlinedTextField(
                value = baseUrl,
                onValueChange = { baseUrl = it },
                modifier = Modifier.fillMaxWidth(),
                placeholder = { Text(suggested) },
                singleLine = true,
                keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.None, autoCorrect = false),
            )
            Spacer(modifier = Modifier.height(16.dp))
            Text("Auth token")
            OutlinedTextField(
                value = token,
                onValueChange = { token = it },
                modifier = Modifier.fillMaxWidth(),
                placeholder = { Text("Paste token") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.None, autoCorrect = false),
            )
            if (error != null) {
                Spacer(modifier = Modifier.height(12.dp))
                Text(error ?: "", modifier = Modifier.fillMaxWidth())
            }
            Spacer(modifier = Modifier.height(24.dp))
            Button(
                onClick = {
                    error = null
                    val b = baseUrl.trim().ifBlank { suggested }
                    val t = token.trim()
                    if (t.isBlank()) {
                        error = "Token required"
                        return@Button
                    }
                    validating = true
                    scope.launch {
                        Preferences.setBaseUrl(b)
                        Preferences.setToken(t)
                        deps.api.healthz().fold(
                            onSuccess = { ok ->
                                if (!ok) {
                                    Preferences.setConfigOk(false)
                                    error = "Health check failed. Verify Base URL and that the host is reachable."
                                } else {
                                    Preferences.setConfigOk(true)
                                    onContinue()
                                }
                            },
                            onFailure = {
                                Preferences.setConfigOk(false)
                                error = it.message ?: "Failed to connect"
                            }
                        )
                        validating = false
                    }
                },
                enabled = !validating,
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(if (validating) "Checking…" else "Continue")
            }
        }
    }
}

