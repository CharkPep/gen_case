// export type BetweenBanksRate = {
//     buy: number,
//     buy_change: number,
//     sell: number,
//     sell_change: number,
//     update_at: number,
// }

export type BankRate = {
    bank: string,
    buy?: number,
    sell?: number,
    buy_online?: number,
    sell_online?: number,
    site_url?: string,
    update_at: Date,
    source_url: string,
}

