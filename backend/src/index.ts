import express from 'express';
import cors from 'cors';
import routes from './routes';
import errorHandler from './middleware/errorHandler';
import path from 'path'; // Import path module
import * as dotenv from 'dotenv';

dotenv.config(); // Load environment variables from .env file

const app = express();
const port = process.env.PORT || 3001;

app.use(cors());
app.use(express.json());

// Serve static files from the 'public' directory
const publicDir = path.join(__dirname, '../../public');
app.use(express.static(publicDir));


app.use('/api', routes);
app.use(errorHandler);

// Handle all other routes by serving the frontend index.html
app.get('*', (req, res) => {
    res.sendFile(path.join(publicDir, 'index.html'));
});


const server = app.listen(port, () => {
    console.log(`Server listening on port ${port}`);
});

const gracefulShutdown = (signal: string) => {
    process.on(signal, async () => {
        console.log(`Received ${signal}, shutting down...`);
        server.close(() => {
            console.log('Server closed.');
            process.exit(0);
        });
    });
};

gracefulShutdown('SIGINT');
gracefulShutdown('SIGTERM');
