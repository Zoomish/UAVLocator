import { Injectable, Logger, OnApplicationBootstrap } from '@nestjs/common'
import { Interval } from '@nestjs/schedule'
import { InjectDataSource, InjectRepository } from '@nestjs/typeorm'
import { randomUUID } from 'crypto'
import { DataSource, LessThan, Repository } from 'typeorm'
import { UserService } from '../user/user.service'
import { DbKeepAlive } from './entities/db-keep-alive.entity'

@Injectable()
export class GetActiveService implements OnApplicationBootstrap {
    private readonly logger = new Logger(GetActiveService.name)

    constructor(
        @InjectRepository(DbKeepAlive)
        private readonly keepAliveRepository: Repository<DbKeepAlive>,
        @InjectDataSource()
        private readonly dataSource: DataSource,
        private readonly userService: UserService
    ) {}

    async onApplicationBootstrap() {
        await this.ping()
    }

    @Interval(1000 * 60 * 3)
    async handleTimeout() {
        await this.ping()
    }

    async ping() {
        await this.dataSource.transaction(async (manager) => {
            const repo = manager.getRepository(DbKeepAlive)

            await repo.save(
                Array.from({ length: 10 }, () =>
                    repo.create({ payload: randomUUID() })
                )
            )

            await repo.delete({
                createdAt: LessThan(new Date(Date.now() - 1000 * 60 * 60 * 24)),
            })

            await repo.count()
        })

        await this.userService.findAll()
        await this.userService.findAllWithLocation('keepalive ping load')

        const admin = await this.userService.findAdmin()
        if (admin) {
            await this.userService.update(admin.tgId, { notifications: false })
            await this.userService.update(admin.tgId, { notifications: true })
        }

        this.logger.log('Database keep-alive load completed')
    }
}
