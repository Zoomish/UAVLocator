import { Injectable, Logger } from '@nestjs/common'
import { UserService } from '../user/user.service'

@Injectable()
export class GetActiveService {
    constructor(private readonly userService: UserService) {}
    private readonly logger = new Logger(GetActiveService.name)

    async handleTimeout() {
        const user = await this.userService.findAdmin()
        if (user) {
            await this.userService.update(user.tgId, { admin: false })
            await this.userService.update(user.tgId, { admin: true })
            this.logger.log('Admin updated')
        }
    }
}
