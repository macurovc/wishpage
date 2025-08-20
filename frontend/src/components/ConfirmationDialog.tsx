import React from 'react';

interface ConfirmationDialogProps {
    isOpen: boolean;
    title: string;
    message: string;
    confirmText: string;
    onConfirm: () => void;
    onCancel: () => void;
}

const ConfirmationDialog: React.FC<ConfirmationDialogProps> = ({ isOpen, title, message, confirmText, onConfirm, onCancel }) => {
    if (!isOpen) {
        return null;
    }

    return (
        <div className="confirmation-dialog-overlay">
            <div className="confirmation-dialog">
                <h3>{title}</h3>
                <p>{message}</p>
                <div className="confirmation-buttons">
                    <button type="button" onClick={onConfirm}>{confirmText}</button>
                    <button type="button" onClick={onCancel}>Cancel</button>
                </div>
            </div>
        </div>
    );
};

export default ConfirmationDialog;
