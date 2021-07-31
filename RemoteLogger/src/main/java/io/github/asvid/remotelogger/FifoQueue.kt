package io.github.asvid.remotelogger.fifo

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch

sealed class Action<V> {
    data class Add<V>(val value: V) : Action<V>()
    data class Pop<V>(val value: CompletableDeferred<V?>) : Action<V>()
}

interface FifoQueue<V> {
    suspend fun pop(): V?
    suspend fun add(value: V)
}

class FifoQueueImpl<V>(
    private val scope: CoroutineScope
) : FifoQueue<V> {

    private val actions = Channel<Action<V>>()
    private val state = Channel<V>()

    init {
        scope.launch {
            for (action in actions) {
                when (action) {
                    is Action.Add -> state.send(action.value)
                    is Action.Pop -> state.receive()
                }
            }
        }
    }

    override suspend fun pop(): V? {
        val valueDeferred = CompletableDeferred<V?>()
        actions.send(Action.Pop(valueDeferred))
        return valueDeferred.await()
    }

    override suspend fun add(value: V) {
        actions.send(Action.Add(value))
    }
}