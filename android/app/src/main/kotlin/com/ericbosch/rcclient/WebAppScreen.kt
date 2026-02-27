package com.ericbosch.rcclient

import android.annotation.SuppressLint
import android.webkit.WebChromeClient
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.runtime.Composable
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView

@SuppressLint("SetJavaScriptEnabled")
@Composable
fun WebAppScreen(onOpenSettings: () -> Unit) {
    val baseUrl = Preferences.getBaseUrl()
    val token = Preferences.getToken()
    val webViewState = remember { mutableStateOf<WebView?>(null) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Remote Control") },
                actions = {
                    IconButton(onClick = onOpenSettings) { Text("Settings") }
                    IconButton(onClick = { webViewState.value?.reload() }) {
                        Icon(Icons.Filled.Refresh, contentDescription = "Refresh")
                    }
                }
            )
        }
    ) { padding ->
        AndroidView(
            modifier = Modifier.fillMaxSize().padding(padding),
            factory = { ctx ->
                WebView(ctx).apply {
                    webViewState.value = this
                    settings.apply {
                        javaScriptEnabled = true
                        domStorageEnabled = true
                        allowFileAccess = false
                        allowContentAccess = false
                        mixedContentMode = WebSettings.MIXED_CONTENT_COMPATIBILITY_MODE
                    }
                    webViewClient = WebViewClient()
                    webChromeClient = WebChromeClient()

                    val escapedToken = token.replace("\\", "\\\\").replace("'", "\\'")
                    val escapedBase = baseUrl.replace("\\", "\\\\").replace("'", "\\'")
                    val escapedUrl = baseUrl.replace("\\", "\\\\").replace("'", "\\'")
                    val html = """
                        <html><head><script>
                        localStorage.setItem('rc-token', '$escapedToken');
                        localStorage.setItem('rc-base', '$escapedBase');
                        window.location.href = '$escapedUrl';
                        </script></head><body>Loading...</body></html>
                    """.trimIndent()

                    loadDataWithBaseURL(baseUrl, html, "text/html", "UTF-8", null)
                }
            },
            update = { wv ->
                webViewState.value = wv
            }
        )
    }
}

