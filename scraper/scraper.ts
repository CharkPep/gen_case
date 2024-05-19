import puppeteer, {Browser, Page} from 'puppeteer';
import * as cheerio from 'cheerio';
import {exitOnError, Logger} from "winston";
import * as path from "node:path";
import {rdb, GetBulkPush, GetPush} from "./queue";
import {BankRate} from "./interface";
const winston = require('winston');


const logPath = "./logs"
const logger: Logger = winston.createLogger({
    level: "info",
    format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.json()
    ),
    defaultMeta: { service: "scraper" },
    transports: [
        process.env?.NODE_ENV === "development" ? new winston.transports.Console({ level: "info"}) : 
        new winston.transports.File({ filename: 'combined.log' }),
        new winston.transports.File({ filename: 'error.log', level: 'error' }),
    ],
});

const LimitLoadedContent = async (page: Page) => {
    await page.setRequestInterception(true)
    page.on('request', (req) => {
        if (
            req.resourceType() == "image" || 
            req.resourceType() == "font" ||
            req.resourceType() == "stylesheet" ||
            req.resourceType() == "script" ||
            req.resourceType() == "media"
        ) {
            req.abort()
            return
        }
        
        req.continue()
    })
}

// const BetweenBanksPrices = async(browser: Browser) => {
//     const page = await browser.newPage()
//     await LimitLoadedContent(page)
//     await page.goto("https://minfin.com.ua/ua/currency/mb/", {
//         waitUntil: 'domcontentloaded',
//         timeout: 60000,
//     })
//     const $ =  cheerio.load(await page.content())
//     const rate = $("table").first()
//     const usd = {} as BetweenBanksRate;
//     const usdRow = rate.find("tbody tr")
//     usd.buy = parseFloat((usdRow.children().eq(1).children().first().contents()[0] as any)?.data.trim().replace(/,/, "."))
//     usd.buy_change = parseFloat(usdRow.children().eq(1).children().children().text())
//     usd.sell = parseFloat((usdRow.children().eq(2).children().first().contents()[0] as any)?.data.trim().replace(/,/, "."))
//     usd.sell_change = parseFloat(usdRow.children().eq(2).children().children().text())
//     return usd;
// }

const BanksPrices = async (browser: Browser) => {
    const page = await browser.newPage()
    await LimitLoadedContent(page)
    await page.goto("https://minfin.com.ua/ua/currency/banks/usd/", {
        waitUntil: 'domcontentloaded',
        timeout: 60000,
    })
    
    const $ =  cheerio.load(await page.content())
    const rateBody = $("#smTable tbody tr")
    const banks: BankRate[] = []
    rateBody.each((idx) => {
        const vals = rateBody.eq(idx).find("[data-title]")
        const bank = {} as BankRate
        bank.bank = vals.eq(0).text().trim()
        bank.buy = parseFloat(vals.eq(1).text().trim())
        bank.sell = parseFloat(vals.eq(2).text().trim())
        bank.buy_online = parseFloat(vals.eq(3).text().trim())
        bank.sell_online = parseFloat(vals.eq(4).text().trim())
        bank.update_at = new Date(vals.eq(5).text().trim())
        bank.site_url = vals.eq(6).children('a').first().attr("href")!
        bank.source_url = "https://minfin.com.ua/ua/currency/banks/usd/"
        banks.push(bank)
    })
    
    return banks;
}

const handler = async () => {
    let browser: Browser = await puppeteer.launch({
            executablePath: process.env?.CHROMIUM ? process.env?.CHROMIUM : "/bin/google-chrome-stable",
            headless : true,
            args: ["--no-sandbox", "--disable-gpu"]
    })

    let bulkQueuePush = GetBulkPush(process.env?.REDIS_STEAM ?? "rate:usd")
    // let queuePush = GetPush("rate:usd:between")
    try {
        const banksPrice  = await BanksPrices(browser)
        logger.info("Successfully scraped")
        await bulkQueuePush(banksPrice)
        // await queuePush(betweenBanks)
    } catch (err) {
        logger.error(err);
        for (const page of (await browser.pages())) {
            await page.screenshot({
                type: 'jpeg',
                path: path.join(logPath, [Date.now(), ".jpeg"].join()),
                quality: 50,
            });
        }

        logger.info("screenshots taken.");
    }

    await browser.close()
}

(async () => {
    let start = Date.now()
    await handler()
    logger.info("Done in: " +  (Date.now() - start).toString() + " ms")
    rdb.disconnect()
})()

