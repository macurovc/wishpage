import nodemailer from 'nodemailer';
import * as dotenv from 'dotenv';

dotenv.config();

const createTransporter = () => {
    const user = process.env.EMAIL_USER;
    const pass = process.env.EMAIL_PASS;
    const host = process.env.EMAIL_HOST || 'smtp.gmail.com';
    const port = parseInt(process.env.EMAIL_PORT || '587', 10);
    const secure = port === 465;

    if (!user || !pass) {
        console.warn('Email service not configured: EMAIL_USER or EMAIL_PASS is missing. Email notifications will be disabled.');
        return null;
    }

    console.log(`Attempting to create email transporter with config: { host: '${host}', port: ${port}, secure: ${secure}, user: '${user}' }`);

    const transporter = nodemailer.createTransport({
        host,
        port,
        secure,
        auth: {
            user,
            pass,
        },
        tls: {
            rejectUnauthorized: false,
        },
    });

    // Verify connection configuration
    transporter.verify((error: Error | null, success: boolean) => {
        if (error) {
            console.error('Email transporter verification failed:', error);
        } else {
            console.log('Email transporter is configured correctly and ready to send messages.');
        }
    });

    return transporter;
};

const transporter = createTransporter();
const recipientEmail = process.env.EMAIL_TO;

export const sendNotificationEmail = async (subject: string, htmlContent: string) => {
    if (!transporter) {
        // This message is now more of a fallback, as createTransporter provides more initial detail.
        console.error('Email transporter is not initialized. Cannot send email.');
        return;
    }
    if (!recipientEmail) {
        console.error('Recipient email (EMAIL_TO) is not configured. Cannot send email.');
        return;
    }

    const mailOptions = {
        from: process.env.EMAIL_FROM || process.env.EMAIL_USER,
        to: recipientEmail,
        subject: `Wishlist Notification: ${subject}`,
        html: htmlContent,
    };

    try {
        const info = await transporter.sendMail(mailOptions);
        console.log(`Notification email sent successfully: ${subject}. Message ID: ${info.messageId}`);
    } catch (error) {
        // Log the detailed error object
        console.error('Error sending notification email:', error);
    }
};
