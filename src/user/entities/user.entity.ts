import { Bot } from 'src/bot/entities/bot.entity'
import {
    Column,
    CreateDateColumn,
    Entity,
    OneToOne,
    PrimaryGeneratedColumn,
    UpdateDateColumn,
} from 'typeorm'

@Entity()
export class User {
    @PrimaryGeneratedColumn()
    id: number

    @Column({ nullable: false, unique: true, type: 'bigint' })
    tgId: number

    @Column({ nullable: false, default: false })
    admin: boolean

    @Column({ nullable: false })
    name: string

    @Column({ nullable: true })
    username: string

    @Column({ nullable: false, array: true, type: 'varchar', default: [] })
    locations: string

    @OneToOne(() => Bot, (bot) => bot.user, {
        cascade: true,
        onDelete: 'CASCADE',
    })
    bot: Bot

    @CreateDateColumn()
    createdAt: Date

    @UpdateDateColumn()
    updatedAt: Date
}
