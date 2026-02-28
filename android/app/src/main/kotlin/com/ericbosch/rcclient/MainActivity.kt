package com.ericbosch.rcclient

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.compose.runtime.CompositionLocalProvider
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.compose.material3.windowsizeclass.ExperimentalMaterial3WindowSizeClassApi
import androidx.compose.material3.windowsizeclass.calculateWindowSizeClass
import com.ericbosch.rcclient.api.RcApiClient
import com.ericbosch.rcclient.di.Deps
import com.ericbosch.rcclient.di.LocalDeps
import com.ericbosch.rcclient.ui.screens.HomeScreen
import com.ericbosch.rcclient.ui.screens.SetupScreen
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient

class MainActivity : ComponentActivity() {
    @OptIn(ExperimentalMaterial3WindowSizeClassApi::class)
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            val deps = run {
                val http = OkHttpClient.Builder().build()
                val json = Json {
                    ignoreUnknownKeys = true
                    isLenient = true
                }
                Deps(http = http, json = json, api = RcApiClient(http, json))
            }
            MaterialTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    CompositionLocalProvider(LocalDeps provides deps) {
                        val navController = rememberNavController()
                        val start = if (Preferences.getToken().isNotEmpty() && Preferences.getBaseUrl().isNotEmpty() && Preferences.isConfigOk()) "home" else "setup"
                        val windowSize = calculateWindowSizeClass(this@MainActivity)
                        NavHost(
                            navController = navController,
                            startDestination = start
                        ) {
                            composable("setup") {
                                SetupScreen(
                                    onContinue = { navController.navigate("home") { popUpTo(0) } }
                                )
                            }
                            composable("settings") {
                                SettingsScreen(
                                    onSaved = { navController.navigate("home") { popUpTo(0) } }
                                )
                            }
                            composable("home") {
                                HomeScreen(
                                    windowSize = windowSize,
                                    onOpenSettings = { navController.navigate("settings") }
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
