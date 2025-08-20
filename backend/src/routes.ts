import express from 'express';
import { getDB } from './db';
import { Database } from 'sqlite';
import AppError from './utils/AppError';
import * as dotenv from 'dotenv';
import { sendNotificationEmail } from './utils/emailService';

dotenv.config();

const router = express.Router();

const verifyPassword = async (req: express.Request, res: express.Response, next: express.NextFunction) => {
    const { password } = req.body;
    const correctPassword = process.env.EDIT_PASSWORD;

    if (!correctPassword) {
        return next(new AppError('Server configuration error: EDIT_PASSWORD not set', 500));
    }

    if (!password || password !== correctPassword) {
        return next(new AppError('Unauthorized: Incorrect password', 401));
    }
    next();
};


// Get all items (no password needed)
router.get('/items', async (req, res, next) => {
    try {
        const db: Database = await getDB();
        const items = await db.all('SELECT items.*, family_members.name as family_member_name FROM items JOIN family_members ON items.family_member_id = family_members.id');
        res.json(items);
    } catch (error) {
        next(new AppError('Failed to fetch items', 500));
    }
});

// Add a new item (protected route - requires authentication)
router.post('/items', verifyPassword, async (req, res, next) => {
  try {
    const { family_member_id, name, link, price } = req.body;

    if (!family_member_id || !name) {
      return next(new AppError('Missing required fields', 400));
    }

    const db = await getDB();
    const result = await db.run(
      'INSERT INTO items (family_member_id, name, link, price) VALUES (?, ?, ?, ?)',
      [family_member_id, name, link, price]
    );

    // Get the newly inserted item with family member name for notification
    const newItem = await db.get('SELECT items.*, family_members.name as family_member_name FROM items JOIN family_members ON items.family_member_id = family_members.id WHERE items.id = ?', result.lastID);

    // Send notification email
    const subject = 'New Item Added';
    const htmlContent = `
        <p>A new item has been added to the wishlist:</p>
        <ul>
            <li><strong>Item:</strong> ${newItem.name}</li>
            <li><strong>For:</strong> ${newItem.family_member_name}</li>
            <li><strong>Price:</strong> ${newItem.price ? `${newItem.price}` : 'N/A'}</li>
            <li><strong>Link:</strong> ${newItem.link ? `<a href="${newItem.link}">${newItem.link}</a>` : 'N/A'}</li>
        </ul>
    `;
    sendNotificationEmail(subject, htmlContent);

    res.status(201).json(newItem); // Return the new item
  } catch (error) {
    next(new AppError('Failed to add item', 500));
  }
});


router.put('/items/:id/reserve', async (req, res, next) => {
    try {
        const { id } = req.params;
        const { reserved } = req.body;

        if (typeof reserved !== 'boolean') {
            return next(new AppError('Invalid reserved value', 400));
        }

        const db = await getDB();

        // First, get the item details for the notification
        const item = await db.get('SELECT items.*, family_members.name as family_member_name FROM items JOIN family_members ON items.family_member_id = family_members.id WHERE items.id = ?', id);

        if (!item) {
            return next(new AppError('Item not found', 404));
        }

        // Then, update the reservation status
        await db.run('UPDATE items SET reserved = ? WHERE id = ?', [reserved, id]);

        // Finally, send the notification email
        const subject = `Item ${reserved ? 'Reserved' : 'Un-reserved'}`;
        const htmlContent = `
            <p>An item's reservation status has been updated:</p>
            <ul>
                <li><strong>Item:</strong> ${item.name}</li>
                <li><strong>For:</strong> ${item.family_member_name}</li>
                <li><strong>Status:</strong> ${reserved ? 'Reserved' : 'Available'}</li>
            </ul>
        `;
        sendNotificationEmail(subject, htmlContent);

        res.json({ message: 'Item reservation status updated' });
    } catch (error) {
        next(new AppError('Failed to update reservation status', 500));
    }
});

// Delete an item (protected route - requires authentication)
router.delete('/items/:id', verifyPassword, async (req, res, next) => {
    try {
        const { id } = req.params;
        const db = await getDB();

        // Get item details before deleting for the notification
        const item = await db.get('SELECT items.*, family_members.name as family_member_name FROM items JOIN family_members ON items.family_member_id = family_members.id WHERE items.id = ?', id);

        if (!item) {
            return next(new AppError('Item not found', 404));
        }

        await db.run('DELETE FROM items WHERE id = ?', id);

        // Send notification email
        const subject = 'Item Deleted';
        const htmlContent = `
            <p>An item has been deleted from the wishlist:</p>
            <ul>
                <li><strong>Item:</strong> ${item.name}</li>
                <li><strong>For:</strong> ${item.family_member_name}</li>
            </ul>
        `;
        sendNotificationEmail(subject, htmlContent);

        res.json({ message: 'Item deleted' });
    } catch (error) {
        next(new AppError('Failed to delete item', 500));
    }
});

// Get all family members (no password needed)
router.get('/family_members', async (req, res, next) => {
    try {
        const db: Database = await getDB();
        // Modified query to order by name alphabetically
        const members = await db.all('SELECT * FROM family_members ORDER BY name ASC');
        res.json(members);
    } catch (error) {
        next(new AppError('Failed to fetch family members', 500));
    }
});

// Add a family member (protected route - requires authentication)
router.post('/family_members', verifyPassword, async (req, res, next) => {
    try {
        const { name } = req.body;
        const db = await getDB();
        const result = await db.run('INSERT INTO family_members (name) VALUES (?)', name);
        const newMember = await db.get('SELECT * FROM family_members WHERE id = ?', result.lastID);

        // Send notification email
        const subject = 'New Family Member Added';
        const htmlContent = `<p>A new family member has been added: <strong>${newMember.name}</strong></p>`;
        sendNotificationEmail(subject, htmlContent);

        res.status(201).json(newMember);
    } catch (error) {
        next(new AppError('Failed to add family member', 500));
    }
});

// Delete a family member (protected route - requires authentication)
router.delete('/family_members/:id', verifyPassword, async (req, res, next) => {
    try {
        const { id } = req.params;
        const db = await getDB();

        // Get member details before deleting for the notification
        const member = await db.get('SELECT * FROM family_members WHERE id = ?', id);

        if (!member) {
            return next(new AppError('Family member not found', 404));
        }

        await db.run('DELETE FROM family_members WHERE id = ?', id);

        // Send notification email
        const subject = 'Family Member Deleted';
        const htmlContent = `<p>A family member has been deleted: <strong>${member.name}</strong></p>`;
        sendNotificationEmail(subject, htmlContent);

        res.json({ message: 'Family member deleted' });
    } catch (error) {
        next(new AppError('Failed to delete family member', 500));
    }
});

// Check password (existing route - no changes needed)
router.post('/check-password', async (req, res) => {
    const { password } = req.body;
    const correctPassword = process.env.EDIT_PASSWORD;

    if (!correctPassword) {
        return res.json({ valid: false });
    }

    if (password === correctPassword) {
        res.json({ valid: true });
    } else {
        res.json({ valid: false });
    }
});

export default router;
