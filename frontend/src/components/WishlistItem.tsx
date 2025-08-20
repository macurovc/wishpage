import React from 'react';
import { Item } from '../types';
import { useStore } from '../store';

interface WishlistItemProps {
  item: Item;
  onReserve: (id: number, reserved: boolean) => void;
  isEditMode: boolean;
  onDelete: (id: number) => void;
}

const WishlistItem: React.FC<WishlistItemProps> = ({ item, onReserve, isEditMode, onDelete }) => {
    const showConfirmation = useStore((state) => state.showConfirmation);

    const handleReserveClick = () => {
        const actionText = item.reserved ? 'Unreserve' : 'Reserve';
        showConfirmation({
            title: `${actionText} "${item.name}"?`,
            message: `Are you sure you want to ${actionText.toLowerCase()} this item?`,
            confirmText: `Yes, ${actionText}`,
            onConfirm: () => onReserve(item.id, !item.reserved),
        });
    };

    const handleDeleteClick = () => {
        showConfirmation({
            title: `Delete "${item.name}"?`,
            message: 'Are you sure you want to delete this item? This action cannot be undone.',
            confirmText: 'Yes, Delete',
            onConfirm: () => onDelete(item.id),
        });
    };

    return (
        <div className={`wishlist-item ${item.reserved ? 'reserved' : ''}`}>
            <div className="item-details">
                <div className="item-name">{item.name}</div>
                <div className="item-meta">
                    {item.price && <span className="item-price">€{item.price}</span>}
                </div>
            </div>
            <div className="item-footer">
                <div className="item-link-container">
                    {item.link && (
                        <a href={item.link} target="_blank" rel="noopener noreferrer" className="item-link">
                            View Item
                            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="feather feather-external-link"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path><polyline points="15 3 21 3 21 9"></polyline><line x1="10" y1="14" x2="21" y2="3"></line></svg>
                        </a>
                    )}
                </div>
                <div className="item-actions">
                    {isEditMode ? (
                        <>
                            <button type="button" onClick={handleDeleteClick} className="delete-button">Delete</button>
                            {item.reserved ? (
                                <button type="button" onClick={handleReserveClick} className="unreserve-button">
                                    Unreserve
                                </button>
                            ) : (
                                <button type="button" onClick={handleReserveClick} className="reserve-button">
                                    Reserve
                                </button>
                            )}
                        </>
                    ) : (
                        <button type="button" onClick={handleReserveClick} disabled={Boolean(item.reserved) && !isEditMode}>
                            {item.reserved ? 'Reserved' : 'Reserve'}
                        </button>
                    )}
                </div>
            </div>
        </div>
    );
};

export default WishlistItem;
