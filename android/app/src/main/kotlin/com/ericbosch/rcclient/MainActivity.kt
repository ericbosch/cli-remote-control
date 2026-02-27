package com.ericbosch.rcclient

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            MaterialTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    val navController = rememberNavController()
                    NavHost(
                        navController = navController,
                        startDestination = "sessions"
                    ) {
                        composable("settings") {
                            SettingsScreen(
                                onSaved = { navController.navigate("sessions") { popUpTo(0) } }
                            )
                        }
                        composable("sessions") {
                            SessionsScreen(
                                onOpenSettings = { navController.navigate("settings") },
                                onAttach = { id -> navController.navigate("terminal/$id") }
                            )
                        }
                        composable("terminal/{sessionId}") { backStackEntry ->
                            val id = backStackEntry.arguments?.getString("sessionId") ?: ""
                            TerminalScreen(sessionId = id, onBack = { navController.popBackStack() })
                        }
                    }
                }
            }
        }
    }
}
