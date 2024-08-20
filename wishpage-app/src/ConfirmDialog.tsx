import React, { useState } from 'react';

interface ConfirmDialogProps {
    isOpen: boolean;
    message: string;
    onConfirm: () => void;
    onCancel: () => void;
}

export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({ isOpen, message, onConfirm, onCancel }) => {
    if (!isOpen) return null;

    return (
        <div className="modal-overlay">
            <div className="modal">
                <p>{message}</p>
                <div className="modal-buttons">
                    <button onClick={onConfirm}>✅ Yes</button>
                    <button onClick={onCancel}>❌ No</button>
                </div>
            </div>
        </div>
    );
};

interface MessageDialogProps {
    isOpen: boolean
    message: string
    onConfirm: () => void
}

export const MessageDialog: React.FC<MessageDialogProps> = ({ isOpen, message, onConfirm }) => {
    if (!isOpen) return null;

    return (
        <div className="modal-overlay">
            <div className="modal">
                <p>{message}</p>
                <div className="modal-buttons">
                    <button onClick={onConfirm}>Ok</button>
                </div>
            </div>
        </div>
    );
};

interface LoginDialogProps {
    isOpen: boolean;
    message: string;
    onLogin: (password: string) => Promise<boolean>;
    onCancel: () => void;
}

export const LoginDialog: React.FC<LoginDialogProps> = ({ isOpen, message, onLogin, onCancel }) => {
    const [password, setPassword] = useState('')
    const [showError, setShowError] = useState(false)
    if (!isOpen) return null;

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        onLogin(password).then(success => setShowError(!success))
        setPassword('')
    }

    return (
        <div className="modal-overlay">
            <div className="modal">
                <p>{message}</p>
                <form onSubmit={handleSubmit}>
                    <input
                        type='password'
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        placeholder='Admin password'
                        required
                    />
                    {showError && <p className='error'>Login error!</p>}
                    <div className="modal-buttons">
                        <button type='submit'>🔑 Login</button>
                        <button onClick={onCancel}>❌ Cancel</button>
                    </div>
                </form>
            </div>
        </div>
    );
};