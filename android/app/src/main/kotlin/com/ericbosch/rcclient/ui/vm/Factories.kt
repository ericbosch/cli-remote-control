package com.ericbosch.rcclient.ui.vm

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import com.ericbosch.rcclient.api.RcApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient

class SessionsViewModelFactory(private val api: RcApiClient) : ViewModelProvider.Factory {
    override fun <T : ViewModel> create(modelClass: Class<T>): T {
        @Suppress("UNCHECKED_CAST")
        return SessionsViewModel(api) as T
    }
}

class SessionViewModelFactory(
    private val http: OkHttpClient,
    private val api: RcApiClient,
    private val json: Json,
) : ViewModelProvider.Factory {
    override fun <T : ViewModel> create(modelClass: Class<T>): T {
        @Suppress("UNCHECKED_CAST")
        return SessionViewModel(http, api, json) as T
    }
}
