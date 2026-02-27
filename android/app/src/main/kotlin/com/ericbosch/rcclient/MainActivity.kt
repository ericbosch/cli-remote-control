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
                    val start = if (Preferences.getToken().isNotEmpty()) "web" else "settings"
                    NavHost(
                        navController = navController,
                        startDestination = start
                    ) {
                        composable("settings") {
                            SettingsScreen(
                                onSaved = { navController.navigate("web") { popUpTo(0) } }
                            )
                        }
                        composable("web") {
                            WebAppScreen(onOpenSettings = { navController.navigate("settings") })
                        }
                    }
                }
            }
        }
    }
}
