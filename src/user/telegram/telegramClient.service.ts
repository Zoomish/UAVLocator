import { Injectable, OnModuleDestroy } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { TelegramClient } from 'telegram'
import { StringSession } from 'telegram/sessions'
import { TelegramLoggerService } from './telegramLogger.service'

@Injectable()
export class TelegramClientService implements OnModuleDestroy {
    private client: TelegramClient | null = null
    private isConnecting = false
    private connectionPromise: Promise<TelegramClient> | null = null

    constructor(
        private readonly configService: ConfigService,
        private readonly telegramLoggerService: TelegramLoggerService
    ) {}

    async getClient(): Promise<TelegramClient> {
        if (this.client && this.client.connected) {
            return this.client
        }

        if (this.isConnecting && this.connectionPromise) {
            return this.connectionPromise
        }

        this.isConnecting = true
        this.connectionPromise = this.createClient()

        try {
            this.client = await this.connectionPromise
            return this.client
        } finally {
            this.isConnecting = false
            this.connectionPromise = null
        }
    }

    private async createClient(): Promise<TelegramClient> {
        const apiId = this.configService.get<string>('API_ID')
        const apiHash = this.configService.get<string>('API_HASH')
        const session = this.configService.get<string>('SESSION')

        const logger = this.telegramLoggerService.getLogger()

        const client = new TelegramClient(
            new StringSession(session),
            parseInt(apiId),
            apiHash,
            {
                connectionRetries: 2,
                timeout: 30000,
                useWSS: true,
                autoReconnect: false,
                baseLogger: logger,
            }
        )

        await client.connect()
        return client
    }

    async disconnect() {
        if (this.client && this.client.connected) {
            try {
                await this.client.disconnect()
            } catch (error) {
                // Игнорируем ошибки при отключении
            }
            this.client = null
        }
    }

    async onModuleDestroy() {
        await this.disconnect()
    }

    isConnected(): boolean {
        return !!(this.client && this.client.connected)
    }
}
