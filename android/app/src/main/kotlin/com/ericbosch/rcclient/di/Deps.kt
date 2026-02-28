package com.ericbosch.rcclient.di

import androidx.compose.runtime.staticCompositionLocalOf
import com.ericbosch.rcclient.api.RcApiClient
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient

data class Deps(
    val http: OkHttpClient,
    val json: Json,
    val api: RcApiClient,
)

val LocalDeps = staticCompositionLocalOf<Deps> {
    error("Deps not provided")
}

