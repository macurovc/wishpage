import React, { useState, useEffect, useRef } from 'react';

interface PasswordDialogProps {
    isOpen: boolean;
    onConfirm: (password: string) => void;
    onCancel: () => void;
    error?: string | null;
}

const PasswordDialog: React.FC<PasswordDialogProps> = ({ isOpen, onConfirm, onCancel, error }) => {
    const [password, setPassword] = useState('');
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (isOpen) {
            // Reset password and focus the input when the dialog opens
            setPassword('');
            setTimeout(() => inputRef.current?.focus(), 100);
        }
    }, [isOpen]);

    if (!isOpen) {
        return null;
    }

    const handleSubmit = (event: React.FormEvent) => {
        event.preventDefault();
        onConfirm(password);
    };

    return (
        <div className="password-dialog-overlay" onClick={onCancel}>
            <div className="password-dialog" onClick={(e) => e.stopPropagation()}>
                <form onSubmit={handleSubmit}>
                    <h3>Enter Edit Mode 🔒</h3>
                    <p>Please enter the password to make changes.</p>
                    <div className="form-group">
                        <label htmlFor="edit-mode-password">Password</label>
                        <input
                            ref={inputRef}
                            id="edit-mode-password"
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            autoComplete="current-password"
                            required
                        />
                    </div>
                    {error && <p className="error-message">{error}</p>}
                    <div className="password-dialog-buttons">
                        <button type="button" onClick={onCancel} className="cancel-button">Cancel</button>
                        <button type="submit" className="confirm-button">Enter</button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default PasswordDialog;