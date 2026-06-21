import { Controller, Get } from '@nestjs/common'
import { GetActiveService } from './tasks/getActiveService.service'

@Controller('app')
export class AppController {
    constructor(private readonly getActiveService: GetActiveService) {}
    @Get()
    async AAA() {
        await this.getActiveService.ping()
        return {
            message: 'Hello World!',
        }
    }
}
