import { IsOptional } from 'class-validator'

export class UpdateUserDto {
    @IsOptional()
    name?: string

    @IsOptional()
    username?: string

    @IsOptional()
    notifications?: boolean

    @IsOptional()
    admin?: boolean

    @IsOptional()
    locations?: string[]

    @IsOptional()
    expires_in?: Date
}
