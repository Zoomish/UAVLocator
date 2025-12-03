import { IsNotEmpty } from 'class-validator'

export class CreateUserDto {
    @IsNotEmpty()
    name: string

    @IsNotEmpty()
    username: string

    @IsNotEmpty()
    tgId: number

    @IsNotEmpty()
    expires_in: Date
}
