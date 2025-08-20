import React from 'react';
import { FamilyMember, Item } from '../types';
import WishlistItem from './WishlistItem';

interface WishlistProps {
    groupedItems: { [familyName: string]: Item[] };
    familyMembers: FamilyMember[];
    onReserve: (id: number, reserved: boolean) => void;
    isEditMode: boolean;
    onDelete: (id: number) => void;
    onSelectFamilyMember: (familyName: string | null) => void;
    selectedFamilyMember: string | null;
}

const Wishlist: React.FC<WishlistProps> = ({ groupedItems, familyMembers, onReserve, isEditMode, onDelete, onSelectFamilyMember, selectedFamilyMember }) => {

    return (
        <div className="wishlist">
            <div className="family-member-buttons">
                {familyMembers.map(member => (
                    <button
                        type="button"
                        key={member.id}
                        className={`family-member-name-button ${selectedFamilyMember === member.name ? 'active' : ''}`}
                        onClick={() => onSelectFamilyMember(member.name)}
                    >
                        <h3>{member.name}</h3>
                    </button>
                ))}
            </div>

            {selectedFamilyMember && familyMembers
                .filter(member => member.name === selectedFamilyMember)
                .map(member => (
                    <div key={member.id} className="person-section">
                        <div className="items-grid">
                            {groupedItems[member.name]?.map(item => (
                                <WishlistItem key={item.id} item={item} onReserve={onReserve} isEditMode={isEditMode} onDelete={onDelete} />
                            ))}
                        </div>
                    </div>
                ))}
        </div>
    );
};

export default Wishlist;
