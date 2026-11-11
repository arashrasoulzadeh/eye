package ir.meetarash.eye

import android.content.Context
import android.os.Handler
import android.os.Looper
import org.json.JSONArray
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
import java.util.UUID
import java.util.concurrent.Executors

data class EyeOptions(
    val endpoint: String,
    val apiKey: String,
    val tenantId: String,
    val batchSize: Int = 20,
    val maxBufferSize: Int = 1000,
    val flushIntervalMs: Long = 5000L,
)

/** Small dependency-free Android client matching the web/Go SDK contract. */
class EyeClient(context: Context, private val options: EyeOptions) {
    private val executor = Executors.newSingleThreadExecutor()
    private val main = Handler(Looper.getMainLooper())
    private val prefs = context.applicationContext.getSharedPreferences("eye", Context.MODE_PRIVATE)
    private val lock = Any()
    private val buffer = ArrayDeque<JSONObject>()
    private var identity = JSONObject()
    private var sessionId: String? = prefs.getString("${options.tenantId}:session", null)
    private var timer: Runnable? = null

    init {
        require(options.endpoint.isNotBlank() && options.apiKey.isNotBlank() && options.tenantId.isNotBlank())
        if (options.flushIntervalMs > 0) {
            val task = object : Runnable { override fun run() { flush(); main.postDelayed(this, options.flushIntervalMs) } }
            timer = task
            main.postDelayed(task, options.flushIntervalMs)
        }
    }

    fun ensureSession(onResult: (Result<String>) -> Unit) {
        synchronized(lock) { sessionId }?.let { onResult(Result.success(it)); return }
        executor.execute {
            runCatching {
                val response = request("/sessions", JSONObject().put("tenant_id", options.tenantId)
                    .put("device", JSONObject().put("platform", "android")), "POST")
                val id = response.optString("ID", response.optString("id"))
                require(id.isNotBlank()) { "session response did not contain ID" }
                synchronized(lock) { sessionId = id }
                prefs.edit().putString("${options.tenantId}:session", id).apply()
                id
            }.onSuccess { main.post { onResult(Result.success(it)) } }
                .onFailure { main.post { onResult(Result.failure(it)) } }
        }
    }

    fun track(name: String, payload: JSONObject? = null, dims: JSONObject? = null): Result<Unit> {
        if (name.isBlank()) return Result.failure(IllegalArgumentException("event name is required"))
        val sid = synchronized(lock) { sessionId }
            ?: return Result.failure(IllegalStateException("call ensureSession first"))
        val eventDims = JSONObject()
        synchronized(lock) {
            val keys = identity.keys()
            while (keys.hasNext()) { val key = keys.next(); eventDims.put(key, identity.get(key)) }
            dims?.let { source -> val sourceKeys = source.keys(); while (sourceKeys.hasNext()) { val key = sourceKeys.next(); eventDims.put(key, source.get(key)) } }
        }
        val event = JSONObject().put("event_id", UUID.randomUUID().toString())
            .put("session_id", sid).put("event_name", name)
            .put("client_ts", System.currentTimeMillis())
            .put("payload", payload ?: JSONObject()).put("dims", eventDims)
        synchronized(lock) {
            if (buffer.size >= options.maxBufferSize) buffer.removeFirst()
            buffer.addLast(event)
            if (buffer.size < options.batchSize) return Result.success(Unit)
        }
        flush()
        return Result.success(Unit)
    }

    fun identify(userId: String, traits: JSONObject = JSONObject()) {
        synchronized(lock) {
            identity = JSONObject()
            identity.put("user_id", userId)
            val keys = traits.keys()
            while (keys.hasNext()) { val key = keys.next(); identity.put(key, traits.get(key)) }
        }
    }

    fun flush() {
        val events = synchronized(lock) {
            if (buffer.isEmpty()) return
            JSONArray(buffer.toList().map { it.toString() })
        }
        executor.execute {
            runCatching { request("/ingest", JSONObject().put("events", events), "POST") }
                .onSuccess { synchronized(lock) { repeat(events.length()) { if (buffer.isNotEmpty()) buffer.removeFirst() } } }
        }
    }

    fun close() { timer?.let(main::removeCallbacks); executor.shutdown() }

    private fun request(path: String, body: JSONObject, method: String): JSONObject {
        val connection = (URL(options.endpoint.trimEnd('/') + path).openConnection() as HttpURLConnection)
        connection.requestMethod = method; connection.connectTimeout = 10000; connection.readTimeout = 10000
        connection.setRequestProperty("content-type", "application/json")
        connection.setRequestProperty("x-api-key", options.apiKey); connection.doOutput = true
        connection.outputStream.use { it.write(body.toString().toByteArray()) }
        val status = connection.responseCode
        val stream = if (status in 200..299) connection.inputStream else connection.errorStream
        val text = stream?.bufferedReader()?.use { it.readText() }.orEmpty()
        if (status !in 200..299) error("Eye request failed: HTTP $status $text")
        return if (text.isBlank()) JSONObject() else JSONObject(text)
    }
}
