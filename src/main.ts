import { ConsoleLogger } from '@nestjs/common'
import { NestFactory } from '@nestjs/core'
import { config } from 'dotenv'
import helmet from 'helmet'
import { AppModule } from './app.module'
config()
async function bootstrap() {
    const app = await NestFactory.create(AppModule, {
        logger: new ConsoleLogger({
            prefix: 'UAVLocator',
            compact: 5,
        }),
    })
    app.setGlobalPrefix('api')
    app.use(helmet())
    app.enableCors({
        origin: (origin, callback) => {
            const allowedOrigins = ['https://web.telegram.org', 'https://t.me']
            if (!origin || allowedOrigins.includes(origin)) {
                callback(null, true)
            } else {
                callback(new Error('Not allowed by CORS'))
            }
        },
        credentials: true,
    })

    await app.listen(3000, async () => {
        console.log(`Server started on port ${await app.getUrl()}`)
    })
}
bootstrap()
