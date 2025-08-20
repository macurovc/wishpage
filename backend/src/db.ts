import sqlite3 from 'sqlite3';
import { open, Database } from 'sqlite';

let db: Database | null = null;

async function initDB(): Promise<Database> {
    if (db) {
        return db;
    }
    const dbFilename = process.env.DATABASE_FILENAME || './wishlist.db'; // Use env variable or default
    db = await open({
        filename: dbFilename,
        driver: sqlite3.Database
    });

    // Database initialization (create tables if they don't exist)
    await db.exec(`
        CREATE TABLE IF NOT EXISTS family_members (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE
        );
    `);

    await db.exec(`
        CREATE TABLE IF NOT EXISTS items (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            family_member_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            link TEXT,
            price REAL,
            reserved BOOLEAN DEFAULT 0,
            FOREIGN KEY (family_member_id) REFERENCES family_members(id)
        );
    `);

    return db;
}

export async function getDB(): Promise<Database> {
    return await initDB();
}

export async function closeDB(): Promise<void> {
    if (db) {
        await db.close();
        db = null;
    }
}
