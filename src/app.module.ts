import { Module } from '@nestjs/common'
import { ConfigModule, ConfigService } from '@nestjs/config'
import { ScheduleModule } from '@nestjs/schedule'
import { ThrottlerModule } from '@nestjs/throttler'
import { TypeOrmModule } from '@nestjs/typeorm'
import { AppController } from './app.controller'
import { BotModule } from './bot/bot.module'
import { GetActiveService } from './tasks/getActiveService.service'
import { UserModule } from './user/user.module'
// eslint-disable-next-line @typescript-eslint/no-require-imports
const fs = require('fs')

@Module({
    imports: [
        ConfigModule.forRoot({ isGlobal: true }),
        ScheduleModule.forRoot(),
        TypeOrmModule.forRootAsync({
            imports: [ConfigModule],
            useFactory: (configService: ConfigService) => ({
                type: 'postgres',
                host: configService.get('POSTGRES_HOST'),
                port: configService.get('POSTGRES_PORT'),
                username: configService.get('POSTGRES_USER'),
                password: configService.get('POSTGRES_PASSWORD'),
                database: configService.get('POSTGRES_DATABASE'),
                entities: [__dirname + '/**/*.entity{.ts,.js}'],
                autoLoadEntities: true,
                synchronize: true,
                migrationsRun: true,
                ssl: {
                    rejectUnauthorized: true,
                    ca: fs.readFileSync('./ca.pem').toString(),
                },
            }),
            inject: [ConfigService],
        }),
        ThrottlerModule.forRoot([
            {
                name: 'short',
                ttl: 1000,
                limit: 10,
            },
            {
                name: 'medium',
                ttl: 10000,
                limit: 50,
            },
            {
                name: 'long',
                ttl: 60000,
                limit: 300,
            },
        ]),
        BotModule,
        UserModule,
    ],
    controllers: [AppController],
    providers: [
        GetActiveService,
    ],
})
export class AppModule {}
