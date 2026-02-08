import { Injectable } from '@nestjs/common'
import { Logger as GramLogger, LogLevel } from 'telegram/extensions/Logger'

@Injectable()
export class TelegramLoggerService {
    private readonly logger: GramLogger

    constructor() {
        this.logger = new GramLogger()
        // Отключаем все уровни логирования
        this.logger.setLevel(LogLevel.NONE)
    }

    getLogger(): GramLogger {
        return this.logger
    }
}
