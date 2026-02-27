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
import androidx.compose.material3.TopAppBar
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView

@SuppressLint("SetJavaScriptEnabled")
@Composable
fun TerminalScreen(sessionId: String, onBack: () -> Unit) {
    val baseUrl = Preferences.getBaseUrl()
    val token = Preferences.getToken()
    val url = "$baseUrl?attach=$sessionId"

    Scaffold(
        topBar = {
            TopAppBar(
                title = { androidx.compose.material3.Text("Terminal") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        }
    ) { padding ->
        AndroidView(
            modifier = Modifier.fillMaxSize().padding(padding),
            factory = { ctx ->
                WebView(ctx).apply {
                    settings.apply {
                        javaScriptEnabled = true
                        domStorageEnabled = true
                        allowFileAccess = false
                        allowContentAccess = false
                        mixedContentMode = WebSettings.MIXED_CONTENT_COMPATIBILITY_MODE
                    }
                    webViewClient = WebViewClient()
                    webChromeClient = WebChromeClient()
                    // Inject token into localStorage then load the terminal URL so the web app can auth
                    val escapedToken = token.replace("\\", "\\\\").replace("'", "\\'")
                    val escapedBase = baseUrl.replace("\\", "\\\\").replace("'", "\\'")
                    val escapedUrl = url.replace("\\", "\\\\").replace("'", "\\'")
                    val html = """
                        <html><head><script>
                        localStorage.setItem('rc-token', '$escapedToken');
                        localStorage.setItem('rc-base', '$escapedBase');
                        window.location.href = '$escapedUrl';
                        </script></head><body>Loading...</body></html>
                    """.trimIndent()
                    loadDataWithBaseURL(baseUrl, html, "text/html", "UTF-8", null)
                }
            }
        )
    }
}
