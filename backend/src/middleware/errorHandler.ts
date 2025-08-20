
import { Request, Response } from 'express';

interface AppError extends Error {
    statusCode?: number;
    isOperational?: boolean;
}

const errorHandler = (err: AppError, req: Request, res: Response) => {
    err.statusCode = err.statusCode || 500;
    err.message = err.message || 'Internal Server Error';

    // Log the error for debugging
    console.error(err);

    // Send a user-friendly error response
    res.status(err.statusCode).json({
        status: 'error',
        statusCode: err.statusCode,
        message: err.isOperational ? err.message : 'An unexpected error occurred. Please try again later.',
    });
};

export default errorHandler;
