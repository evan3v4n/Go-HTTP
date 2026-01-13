import { useState, useEffect, useCallback, useRef } from 'react'

interface Stats {
  numbers: number[]
  processed_logs: number
}

interface QueueStatus {
  queue_depth: number
  queue_capacity: number
  active_workers: number
  total_workers: number
  processed_logs: number
}

interface JobNotification {
  request_id: string
  type: string
  success: boolean
  data?: unknown
  timestamp: string
  duration: string
}

function App() {
  const [inputValue, setInputValue] = useState('')
  const [stats, setStats] = useState<Stats>({ numbers: [], processed_logs: 0 })
  const [queueStatus, setQueueStatus] = useState<QueueStatus>({
    queue_depth: 0,
    queue_capacity: 100,
    active_workers: 0,
    total_workers: 4,
    processed_logs: 0
  })
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastAdded, setLastAdded] = useState<number | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [sseConnected, setSseConnected] = useState(false)
  const [recentJobs, setRecentJobs] = useState<JobNotification[]>([])
  
  const eventSourceRef = useRef<EventSource | null>(null)

  // Setup SSE connection for job notifications
  useEffect(() => {
    const eventSource = new EventSource('/events')
    eventSourceRef.current = eventSource

    eventSource.onopen = () => {
      setSseConnected(true)
      console.log('SSE connected')
    }

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'connected') {
          console.log(`Connected to server with ${data.workers} workers`)
        } else if (data.request_id) {
          // Job notification
          const notification: JobNotification = data
          setRecentJobs(prev => [notification, ...prev].slice(0, 10))
          
          // Auto-refresh stats on job completion
          if (notification.type === 'add' || notification.type === 'clear') {
            fetchStats()
          }
        }
      } catch (err) {
        console.error('Failed to parse SSE message:', err)
      }
    }

    eventSource.onerror = () => {
      setSseConnected(false)
      console.log('SSE disconnected')
    }

    return () => {
      eventSource.close()
    }
  }, [])

  // Fetch stats from the server
  const fetchStats = useCallback(async () => {
    try {
      const response = await fetch('/stats')
      if (!response.ok) throw new Error('Failed to fetch stats')
      const data = await response.json()
      setStats(data)
      setIsConnected(true)
      setError(null)
    } catch (err) {
      setIsConnected(false)
      console.error('Failed to fetch stats:', err)
    }
  }, [])

  // Fetch queue status
  const fetchQueueStatus = useCallback(async () => {
    try {
      const response = await fetch('/queue-status')
      if (!response.ok) throw new Error('Failed to fetch queue status')
      const data = await response.json()
      setQueueStatus(data)
    } catch (err) {
      console.error('Failed to fetch queue status:', err)
    }
  }, [])

  // Poll for stats and queue status
  useEffect(() => {
    fetchStats()
    fetchQueueStatus()
    const statsInterval = setInterval(fetchStats, 1000)
    const queueInterval = setInterval(fetchQueueStatus, 500)
    return () => {
      clearInterval(statsInterval)
      clearInterval(queueInterval)
    }
  }, [fetchStats, fetchQueueStatus])

  // Send a number to the server
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    const num = parseInt(inputValue, 10)
    if (isNaN(num)) {
      setError('Please enter a valid number')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const response = await fetch('/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ number: num }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to add number')
      }
      
      setLastAdded(num)
      setInputValue('')
      
      // Clear the "last added" highlight after animation
      setTimeout(() => setLastAdded(null), 1000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send number to server')
      console.error(err)
    } finally {
      setIsLoading(false)
    }
  }

  // Clear all numbers
  const handleClear = async () => {
    try {
      const response = await fetch('/clear', { method: 'POST' })
      if (!response.ok) throw new Error('Failed to clear')
    } catch (err) {
      setError('Failed to clear numbers')
      console.error(err)
    }
  }

  // Calculate statistics
  const sum = stats.numbers.reduce((a, b) => a + b, 0)
  const avg = stats.numbers.length > 0 ? (sum / stats.numbers.length).toFixed(2) : '0'
  const max = stats.numbers.length > 0 ? Math.max(...stats.numbers) : 0
  const min = stats.numbers.length > 0 ? Math.min(...stats.numbers) : 0

  // Queue depth percentage
  const queuePercent = (queueStatus.queue_depth / queueStatus.queue_capacity) * 100

  return (
    <div className="min-h-screen mesh-bg">
      {/* Header */}
      <header className="border-b border-cyan-900/30 bg-midnight-950/50 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-electric-400 to-electric-600 flex items-center justify-center shadow-lg shadow-electric-500/30">
                <svg className="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <div>
                <h1 className="font-display text-xl font-semibold text-white tracking-tight">
                  Go HTTP Dashboard
                </h1>
                <p className="text-sm text-slate-400 font-light">
                  Worker Pool Architecture Demo
                </p>
              </div>
            </div>
            
            {/* Connection Status */}
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <div className={`w-2.5 h-2.5 rounded-full ${sseConnected ? 'bg-purple-400 status-dot' : 'bg-slate-500'}`} />
                <span className={`text-sm font-mono ${sseConnected ? 'text-purple-400' : 'text-slate-500'}`}>
                  SSE
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div className={`w-2.5 h-2.5 rounded-full ${isConnected ? 'bg-emerald-400 status-dot' : 'bg-red-400'}`} />
                <span className={`text-sm font-mono ${isConnected ? 'text-emerald-400' : 'text-red-400'}`}>
                  {isConnected ? 'Connected' : 'Disconnected'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        {/* Worker Pool Status Bar */}
        <div className="glass-card rounded-2xl p-4 mb-6 animate-fade-in">
          <div className="flex items-center justify-between gap-8">
            {/* Queue Depth */}
            <div className="flex-1">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-slate-400 font-mono">httpRequestsChannel</span>
                <span className="text-sm font-mono text-white">
                  {queueStatus.queue_depth} / {queueStatus.queue_capacity}
                </span>
              </div>
              <div className="h-3 bg-midnight-950/80 rounded-full overflow-hidden">
                <div 
                  className={`h-full rounded-full transition-all duration-300 ${
                    queuePercent > 80 ? 'bg-red-500' : 
                    queuePercent > 50 ? 'bg-amber-500' : 
                    'bg-electric-500'
                  }`}
                  style={{ width: `${Math.max(queuePercent, 2)}%` }}
                />
              </div>
            </div>

            {/* Workers */}
            <div className="flex items-center gap-3">
              <span className="text-sm text-slate-400">Workers:</span>
              <div className="flex gap-1.5">
                {Array.from({ length: queueStatus.total_workers }).map((_, i) => (
                  <div
                    key={i}
                    className={`w-8 h-8 rounded-lg flex items-center justify-center font-mono text-xs transition-all duration-200 ${
                      i < queueStatus.active_workers
                        ? 'bg-emerald-500/30 border border-emerald-500/50 text-emerald-400'
                        : 'bg-midnight-900/50 border border-cyan-900/30 text-slate-500'
                    }`}
                  >
                    {i + 1}
                  </div>
                ))}
              </div>
              <span className="text-sm font-mono text-emerald-400">
                {queueStatus.active_workers} active
              </span>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          
          {/* Left Column - Input & Stats */}
          <div className="space-y-6">
            {/* Input Card */}
            <div className="glass-card rounded-2xl p-6 animate-fade-in">
              <h2 className="font-display text-lg font-medium text-white mb-4 flex items-center gap-2">
                <span className="w-1.5 h-1.5 rounded-full bg-electric-400" />
                Send Number
              </h2>
              
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="relative">
                  <input
                    type="number"
                    value={inputValue}
                    onChange={(e) => setInputValue(e.target.value)}
                    placeholder="Enter a number..."
                    className="w-full px-4 py-3 rounded-xl bg-midnight-950/80 border border-cyan-900/40 
                             text-white placeholder-slate-500 font-mono text-lg
                             focus:outline-none focus:border-electric-500 focus:ring-2 focus:ring-electric-500/20
                             transition-all duration-200"
                  />
                </div>
                
                <button
                  type="submit"
                  disabled={isLoading || !inputValue}
                  className="w-full py-3 px-6 rounded-xl font-display font-medium
                           bg-gradient-to-r from-electric-500 to-electric-600
                           hover:from-electric-400 hover:to-electric-500
                           disabled:opacity-50 disabled:cursor-not-allowed
                           text-white shadow-lg shadow-electric-500/30
                           hover:shadow-electric-400/40
                           transform hover:scale-[1.02] active:scale-[0.98]
                           transition-all duration-200"
                >
                  {isLoading ? (
                    <span className="flex items-center justify-center gap-2">
                      <svg className="animate-spin w-5 h-5" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                      </svg>
                      Queuing...
                    </span>
                  ) : (
                    'Add to Queue →'
                  )}
                </button>
              </form>

              {error && (
                <div className="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
                  {error}
                </div>
              )}
            </div>

            {/* Stats Cards */}
            <div className="grid grid-cols-2 gap-4">
              <StatCard
                label="Total Count"
                value={stats.numbers.length}
                icon="📊"
                color="electric"
              />
              <StatCard
                label="Sum"
                value={sum}
                icon="➕"
                color="amber"
              />
              <StatCard
                label="Average"
                value={avg}
                icon="📈"
                color="purple"
              />
              <StatCard
                label="Range"
                value={stats.numbers.length > 0 ? `${min} → ${max}` : '—'}
                icon="↔️"
                color="emerald"
              />
            </div>
          </div>

          {/* Middle Column - Numbers Display */}
          <div className="lg:col-span-1">
            <div className="glass-card rounded-2xl p-6 h-full animate-fade-in" style={{ animationDelay: '0.1s' }}>
              <div className="flex items-center justify-between mb-4">
                <h2 className="font-display text-lg font-medium text-white flex items-center gap-2">
                  <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
                  mainNumsArr
                </h2>
                <button
                  onClick={handleClear}
                  disabled={stats.numbers.length === 0}
                  className="text-sm px-3 py-1.5 rounded-lg text-slate-400 hover:text-red-400 
                           hover:bg-red-500/10 disabled:opacity-50 disabled:cursor-not-allowed
                           transition-all duration-200 font-mono"
                >
                  Clear All
                </button>
              </div>

              <div className="bg-midnight-950/60 rounded-xl p-4 min-h-[300px] max-h-[400px] overflow-y-auto">
                {stats.numbers.length === 0 ? (
                  <div className="h-full flex flex-col items-center justify-center text-slate-500">
                    <svg className="w-16 h-16 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                    <p className="font-light">Array is empty</p>
                    <p className="text-sm opacity-60">Add numbers to see them here</p>
                  </div>
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {stats.numbers.map((num, index) => (
                      <span
                        key={`${index}-${num}`}
                        className={`number-pill px-3 py-1.5 rounded-lg font-mono text-sm
                                  ${num === lastAdded 
                                    ? 'bg-electric-500 text-white shadow-lg shadow-electric-500/40' 
                                    : 'bg-midnight-800/80 text-slate-300 border border-cyan-900/30'
                                  }
                                  transition-all duration-200`}
                        style={{ animationDelay: `${index * 0.02}s` }}
                      >
                        {num}
                      </span>
                    ))}
                  </div>
                )}
              </div>

              {/* Array representation */}
              <div className="mt-4 p-3 rounded-lg bg-midnight-950/60 font-mono text-xs text-slate-400 overflow-x-auto">
                <span className="text-electric-400">[]int</span>
                {'{'}
                <span className="text-amber-400">{stats.numbers.slice(0, 10).join(', ')}</span>
                {stats.numbers.length > 10 && <span className="text-slate-500">...+{stats.numbers.length - 10} more</span>}
                {'}'}
              </div>
            </div>
          </div>

          {/* Right Column - Worker Pool & Logs */}
          <div className="lg:col-span-1 space-y-6">
            {/* Worker Pool Status */}
            <div className="glass-card rounded-2xl p-6 animate-fade-in" style={{ animationDelay: '0.2s' }}>
              <h2 className="font-display text-lg font-medium text-white mb-4 flex items-center gap-2">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                Worker Pool
              </h2>

              {/* Worker Status Panel */}
              <div className="bg-midnight-950/60 rounded-xl p-4 mb-4">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm text-slate-400">Dispatcher Status</span>
                  <span className="flex items-center gap-2 text-emerald-400 text-sm font-mono">
                    <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                    ACTIVE
                  </span>
                </div>
                
                <div className="space-y-3">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-slate-400">Processed Logs</span>
                    <span className="font-mono text-2xl font-semibold text-white">
                      {queueStatus.processed_logs.toLocaleString()}
                    </span>
                  </div>
                  
                  {/* Progress visualization */}
                  <div className="h-2 bg-midnight-800 rounded-full overflow-hidden">
                    <div 
                      className="h-full bg-gradient-to-r from-electric-500 to-emerald-400 rounded-full transition-all duration-500"
                      style={{ width: `${Math.min((queueStatus.processed_logs % 100), 100)}%` }}
                    />
                  </div>
                </div>
              </div>

              {/* Architecture Diagram Mini */}
              <div className="p-4 rounded-xl border border-dashed border-cyan-900/40">
                <p className="text-xs text-slate-500 text-center mb-3">Data Flow</p>
                <div className="flex items-center justify-center gap-1 text-xs font-mono flex-wrap">
                  <span className="px-2 py-1 rounded bg-blue-500/20 text-blue-400">Entry</span>
                  <span className="text-slate-500">→</span>
                  <span className="px-2 py-1 rounded bg-electric-500/20 text-electric-400">Queue</span>
                  <span className="text-slate-500">→</span>
                  <span className="px-2 py-1 rounded bg-amber-500/20 text-amber-400">Worker</span>
                  <span className="text-slate-500">→</span>
                  <span className="px-2 py-1 rounded bg-emerald-500/20 text-emerald-400">Log</span>
                  <span className="text-slate-500">→</span>
                  <span className="px-2 py-1 rounded bg-purple-500/20 text-purple-400">File</span>
                </div>
              </div>
            </div>

            {/* Recent Jobs via SSE */}
            <div className="glass-card rounded-2xl p-6 animate-fade-in" style={{ animationDelay: '0.3s' }}>
              <h2 className="font-display text-lg font-medium text-white mb-4 flex items-center gap-2">
                <span className="w-1.5 h-1.5 rounded-full bg-purple-400" />
                Live Job Feed
                <span className="ml-auto text-xs font-mono text-slate-500">via SSE</span>
              </h2>

              <div className="bg-midnight-950/60 rounded-xl p-3 max-h-[200px] overflow-y-auto">
                {recentJobs.length === 0 ? (
                  <div className="text-center text-slate-500 text-sm py-4">
                    <p>Waiting for jobs...</p>
                    <p className="text-xs opacity-60 mt-1">Events stream in real-time</p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {recentJobs.map((job, i) => (
                      <div 
                        key={job.request_id + i}
                        className={`p-2 rounded-lg text-xs font-mono ${
                          job.success 
                            ? 'bg-emerald-500/10 border border-emerald-500/20' 
                            : 'bg-red-500/10 border border-red-500/20'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className={`font-semibold ${job.success ? 'text-emerald-400' : 'text-red-400'}`}>
                            {job.type.toUpperCase()}
                          </span>
                          <span className="text-slate-500">{job.duration}</span>
                        </div>
                        <div className="text-slate-400 text-[10px] truncate mt-1">
                          {job.request_id}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-cyan-900/30 mt-auto">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between text-sm text-slate-500">
            <span className="font-mono">go-http-server:8080</span>
            <span className="flex items-center gap-4">
              <span className="flex items-center gap-2">
                <span className="text-amber-400">{queueStatus.total_workers}</span> workers
              </span>
              <span className="flex items-center gap-2">
                <span className="w-1.5 h-1.5 rounded-full bg-electric-400" />
                Worker Pool Edition
              </span>
            </span>
          </div>
        </div>
      </footer>
    </div>
  )
}

// Stat Card Component
interface StatCardProps {
  label: string
  value: string | number
  icon: string
  color: 'electric' | 'amber' | 'purple' | 'emerald'
}

function StatCard({ label, value, icon, color }: StatCardProps) {
  const colorClasses = {
    electric: 'from-electric-500/20 to-electric-600/10 border-electric-500/30',
    amber: 'from-amber-500/20 to-amber-600/10 border-amber-500/30',
    purple: 'from-purple-500/20 to-purple-600/10 border-purple-500/30',
    emerald: 'from-emerald-500/20 to-emerald-600/10 border-emerald-500/30',
  }

  return (
    <div className={`rounded-xl p-4 bg-gradient-to-br ${colorClasses[color]} border backdrop-blur-sm`}>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs text-slate-400 uppercase tracking-wider mb-1">{label}</p>
          <p className="font-mono text-xl font-semibold text-white">{value}</p>
        </div>
        <span className="text-xl">{icon}</span>
      </div>
    </div>
  )
}

export default App
