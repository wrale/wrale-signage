/**
 * IndexedDB-based offline storage for health events with prioritized recovery
 */

import { openDB, DBSchema, IDBPDatabase } from 'idb'
import { ContentEvent, ContentEventType } from '../types'

// Define schema for our IndexedDB
interface HealthDB extends DBSchema {
  events: {
    key: string // timestamp:url:type
    value: StoredEvent
    indexes: {
      'by-priority': number
      'by-timestamp': number
      'by-type': string
    }
  }
  metadata: {
    key: string
    value: {
      lastFlushTime: number
      lastSyncTime: number
      networkStatus: 'online' | 'offline'
      storageQuota: number
      eventCount: number
    }
  }
}

interface StoredEvent {
  event: ContentEvent
  priority: number
  attempts: number
  stored: number
}

export class OfflineStore {
  private db?: IDBPDatabase<HealthDB>
  private dbName = 'health-events'
  private version = 1
  private maxRetries = 3
  private maxAge = 7 * 24 * 60 * 60 * 1000 // 7 days
  private maxEvents = 10000
  private maintenanceInterval?: number
  private networkCheckInterval?: number

  // Event priorities (higher = more important)
  private priorities: Record<ContentEventType, number> = {
    'CONTENT_ERROR': 100,
    'CONTENT_LOADED': 50,
    'CONTENT_VISIBLE': 40,
    'CONTENT_HIDDEN': 30,
    'CONTENT_INTERACTIVE': 35
  }

  // Network status tracking
  private online = navigator.onLine
  private lastOnline = Date.now()
  private backoffDelay = 1000 // Start with 1s

  constructor(
    private readonly onRecover: (events: ContentEvent[]) => Promise<void>
  ) {}

  async initialize() {
    this.db = await openDB<HealthDB>(this.dbName, this.version, {
      upgrade: (db) => this.upgradeDB(db),
      blocked: () => console.warn('Health event DB blocked'),
      blocking: () => console.warn('Health event DB blocking'),
    })

    // Start background tasks
    this.startMaintenance()
    this.trackNetworkStatus()
  }

  private upgradeDB(db: IDBPDatabase<HealthDB>) {
    // Events store with indexes
    const eventStore = db.createObjectStore('events', {
      keyPath: 'key'
    })
    eventStore.createIndex('by-priority', 'priority')
    eventStore.createIndex('by-timestamp', 'stored')
    eventStore.createIndex('by-type', 'event.type')

    // Metadata store
    db.createObjectStore('metadata')
  }

  /**
   * Store health event for offline resilience
   */
  async storeEvent(event: ContentEvent): Promise<void> {
    if (!this.db) throw new Error('Database not initialized')

    const key = this.createEventKey(event)
    const priority = this.priorities[event.type] || 0

    const storedEvent: StoredEvent = {
      event,
      priority,
      attempts: 0,
      stored: Date.now()
    }

    await this.db.put('events', storedEvent)

    // Update metadata
    const tx = this.db.transaction('metadata', 'readwrite')
    const metadata = await tx.store.get('stats') || {
      eventCount: 0,
      lastFlushTime: Date.now(),
      networkStatus: this.online ? 'online' : 'offline'
    }
    metadata.eventCount++
    await tx.store.put(metadata, 'stats')

    // Try recovery if we're online
    if (this.online) {
      this.attemptRecovery()
    }
  }

  /**
   * Get stored events for recovery, ordered by priority
   */
  async getEventsForRecovery(limit = 50): Promise<StoredEvent[]> {
    if (!this.db) throw new Error('Database not initialized')

    const events: StoredEvent[] = []
    let cursor = await this.db.transaction('events')
      .store
      .index('by-priority')
      .openCursor(null, 'prev') // Highest priority first

    while (cursor && events.length < limit) {
      if (cursor.value.attempts < this.maxRetries) {
        events.push(cursor.value)
      }
      cursor = await cursor.continue()
    }

    return events
  }

  /**
   * Mark events as successfully synced
   */
  async markEventsSynced(events: StoredEvent[]): Promise<void> {
    if (!this.db) throw new Error('Database not initialized')

    const tx = this.db.transaction(['events', 'metadata'], 'readwrite')
    
    // Remove synced events
    for (const event of events) {
      await tx.objectStore('events').delete(this.createEventKey(event.event))
    }

    // Update metadata
    const metadata = await tx.objectStore('metadata').get('stats') || {
      eventCount: 0,
      lastSyncTime: Date.now()
    }
    metadata.eventCount = Math.max(0, metadata.eventCount - events.length)
    metadata.lastSyncTime = Date.now()
    await tx.objectStore('metadata').put(metadata, 'stats')

    // Reset backoff on success
    this.backoffDelay = 1000
  }

  /**
   * Mark events as failed to sync
   */
  async markEventsFailed(events: StoredEvent[]): Promise<void> {
    if (!this.db) throw new Error('Database not initialized')

    const tx = this.db.transaction('events', 'readwrite')
    
    for (const event of events) {
      event.attempts++
      if (event.attempts < this.maxRetries) {
        await tx.store.put(event)
      } else {
        // Drop events that have failed too many times
        await tx.store.delete(this.createEventKey(event.event))
      }
    }

    // Increase backoff
    this.backoffDelay = Math.min(this.backoffDelay * 2, 60000) // Max 1 minute
  }

  /**
   * Attempt to recover stored events
   */
  private async attemptRecovery(): Promise<void> {
    if (!this.online) return

    try {
      const events = await this.getEventsForRecovery()
      if (events.length === 0) return

      // Try to sync with server
      await this.onRecover(events.map(e => e.event))
      
      // Success - mark events as synced
      await this.markEventsSynced(events)

    } catch (err) {
      console.error('Recovery attempt failed:', err)
      const events = await this.getEventsForRecovery()
      await this.markEventsFailed(events)

      // Schedule next attempt with backoff
      setTimeout(() => {
        this.attemptRecovery()
      }, this.backoffDelay)
    }
  }

  /**
   * Track network status for recovery
   */
  private trackNetworkStatus() {
    window.addEventListener('online', () => {
      this.online = true
      this.lastOnline = Date.now()
      this.attemptRecovery()
    })

    window.addEventListener('offline', () => {
      this.online = false
    })

    // Also poll network status
    this.networkCheckInterval = window.setInterval(() => {
      const wasOnline = this.online
      this.online = navigator.onLine

      if (!wasOnline && this.online) {
        this.lastOnline = Date.now()
        this.attemptRecovery()
      }
    }, 30000) // Check every 30s
  }

  /**
   * Regular maintenance tasks
   */
  private startMaintenance() {
    this.maintenanceInterval = window.setInterval(async () => {
      if (!this.db) return

      try {
        // Prune old events
        const tx = this.db.transaction('events', 'readwrite')
        let cursor = await tx.store.index('by-timestamp').openCursor()
        const now = Date.now()

        while (cursor) {
          if (now - cursor.value.stored > this.maxAge) {
            await cursor.delete()
          }
          cursor = await cursor.continue()
        }

        // Check quota
        const metadata = await this.db.get('metadata', 'stats')
        if (metadata?.eventCount > this.maxEvents) {
          // Keep only high priority events
          cursor = await tx.store.index('by-priority').openCursor()
          let deleted = 0
          const toDelete = metadata.eventCount - this.maxEvents

          while (cursor && deleted < toDelete) {
            if (cursor.value.priority < 50) { // Keep errors and important events
              await cursor.delete()
              deleted++
            }
            cursor = await cursor.continue()
          }

          metadata.eventCount -= deleted
          await this.db.put('metadata', metadata, 'stats')
        }

      } catch (err) {
        console.error('Maintenance error:', err)
      }
    }, 60 * 60 * 1000) // Run hourly
  }

  private createEventKey(event: ContentEvent): string {
    return `${event.timestamp}:${event.contentUrl}:${event.type}`
  }

  async dispose() {
    if (this.maintenanceInterval) {
      window.clearInterval(this.maintenanceInterval)
    }
    if (this.networkCheckInterval) {
      window.clearInterval(this.networkCheckInterval)
    }
    if (this.db) {
      this.db.close()
    }
  }
}
