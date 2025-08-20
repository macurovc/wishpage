import React, { useState } from 'react';
import { Item, FamilyMember } from '../types';

interface EditControlsProps {
    onAddItem: (item: Omit<Item, 'id' | 'family_member_name' | 'reserved'>) => Promise<void>;
    familyMembers: FamilyMember[];
    onAddFamilyMember: (name: string) => Promise<void>;
    onDeleteFamilyMember: (id: number) => Promise<void>; // Add onDeleteFamilyMember prop
}

interface InputErrors {
    family_member_id?: string;
    name?: string;
    link?: string; // Added link to InputErrors
}

const EditControls: React.FC<EditControlsProps> = ({ onAddItem, familyMembers, onAddFamilyMember, onDeleteFamilyMember }) => {
    const [newFamilyMemberName, setNewFamilyMemberName] = useState('');
    const [familyMemberToDeleteId, setFamilyMemberToDeleteId] = useState<number | null>(null); // State for family member to delete
    const [newItem, setNewItem] = useState<Omit<Item, 'id' | 'family_member_name' | 'reserved'>>({
        family_member_id: 0, // Default to the first family member
        name: '',
        link: '',
        price: null, // Default price as null
    });
    const [localError, setLocalError] = useState<string | null>(null); // Local error state for EditControls
    const [inputErrors, setInputErrors] = useState<InputErrors>({}); // State for input validation errors


    const handleEditControlsAddFamilyMember = () => {
        onAddFamilyMember(newFamilyMemberName);
        setNewFamilyMemberName('');
        setLocalError(null); // Clear any local errors
    };

    const handleEditControlsDeleteFamilyMember = () => {
        if (familyMemberToDeleteId !== null) {
            onDeleteFamilyMember(familyMemberToDeleteId);
            setFamilyMemberToDeleteId(null); // Reset after deletion
            setLocalError(null); // Clear any local errors
        } else {
            setLocalError('Please select a Family Member ID to delete.');
        }
    };


    const handleAddItemClick = async () => {
        // Client-side validation
        const errors: InputErrors = {};
        if (newItem.family_member_id === 0) {
            errors.family_member_id = 'Please select a family member';
        }
        if (!newItem.name.trim()) {
            errors.name = 'Item name is required';
        }
        if (newItem.link && newItem.link.trim() && !isValidUrl(newItem.link)) {
            errors.link = 'Link must be a valid URL';
        }


        if (Object.keys(errors).length > 0) {
            setInputErrors(errors);
            return; // Stop submission if there are errors
        }

        setInputErrors({}); // Clear previous errors if validation passes

        try {
            await onAddItem(newItem);
            setNewItem({
                family_member_id: 0,
                name: '',
                link: '',
                price: null, // Reset price to null
            });
            setLocalError(null); // Clear any local errors on success
        } catch (error) {
            if (error instanceof Error) {
                setLocalError(error.message);
            } else {
                setLocalError('Failed to add item.');
            }
        }
    };


    const isValidUrl = (url: string) => {
        try {
            new URL(url);
            return true;
        } catch {
            return false;
        }
    }


    return (
        <form className="edit-controls" onSubmit={(e) => e.preventDefault()}>
            <h3>Add Family Member 👨‍👩‍👧‍👦</h3>
            <input
                type="text"
                value={newFamilyMemberName}
                onChange={(e) => setNewFamilyMemberName(e.target.value)}
                placeholder="Family Member Name"
                aria-label="Family Member Name" // Accessibility label
                autoComplete="off"
            />
            <button type="button" onClick={handleEditControlsAddFamilyMember}>Add Family Member</button>

            <h3>Delete Family Member 👨‍👩‍👧‍👦</h3>
            <div className="form-group">
                <label htmlFor="familyMemberToDeleteSelect">Select Member to Delete 🗑️</label>
                <select
                    id="familyMemberToDeleteSelect"
                    value={familyMemberToDeleteId === null ? '' : familyMemberToDeleteId}
                    onChange={(e) => setFamilyMemberToDeleteId(e.target.value === '' ? null : parseInt(e.target.value))}
                    aria-label="Select Family Member to Delete" // Accessibility label
                    autoComplete="off"
                >
                    <option value="" disabled>Select Family Member ID</option>
                    {familyMembers.map(member => (
                        <option key={member.id} value={member.id}>{member.name} (ID: {member.id})</option>
                    ))}
                </select>
            </div>
            <button type="button" onClick={handleEditControlsDeleteFamilyMember} disabled={familyMemberToDeleteId === null}>
                Delete Family Member
            </button>


            <h3>Add Item 🎁</h3>
            <div className="form-group">
                <label htmlFor="familyMemberSelect">For whom? 👪</label>
                <select
                    id="familyMemberSelect"
                    value={newItem.family_member_id}
                    onChange={(e) => setNewItem({ ...newItem, family_member_id: parseInt(e.target.value) })}
                    aria-label="Select Family Member" // Accessibility label
                    autoComplete="off"
                >
                    <option value={0} disabled>Select Family Member</option>
                    {familyMembers.map(member => (
                        <option key={member.id} value={member.id}>{member.name}</option>
                    ))}
                </select>
                {inputErrors.family_member_id && <p className="input-error">{inputErrors.family_member_id}</p>}
            </div>
            <div className="form-group">
                <label htmlFor="itemName">Item Name ✨</label>
                <input
                    type="text"
                    id="itemName"
                    value={newItem.name}
                    onChange={(e) => setNewItem({ ...newItem, name: e.target.value })}
                    placeholder="e.g., Lego Star Wars Set"
                    aria-label="Item Name" // Accessibility label
                    autoComplete="off"
                />
                 {inputErrors.name && <p className="input-error">{inputErrors.name}</p>}
            </div>
            <div className="form-group">
                <label htmlFor="itemLink">Link to item 🔗 (optional)</label>
                <input
                    type="text"
                    id="itemLink"
                    value={newItem.link || ''}
                    onChange={(e) => setNewItem({ ...newItem, link: e.target.value })}
                    placeholder="e.g., https://www.example.com/lego-set"
                    aria-label="Item Link" // Accessibility label
                    autoComplete="off"
                />
                {inputErrors.link && <p className="input-error">{inputErrors.link}</p>}
            </div>
            <div className="form-group">
                <label htmlFor="itemPrice">Price (€) 💰 (optional)</label>
                <input
                    type="number" // Changed to type number
                    id="itemPrice"
                    value={newItem.price === null ? '' : newItem.price} // Handle null for number input
                    onChange={(e) => setNewItem({ ...newItem, price: e.target.value === '' ? null : parseFloat(e.target.value) })} // Parse to float, handle empty string
                    placeholder="e.g., 59.99"
                    aria-label="Item Price" // Accessibility label
                    autoComplete="off"
                />
            </div>
            <button type="button" onClick={handleAddItemClick}>Add Item</button>
            {localError && <p className="error-message">{localError}</p>}
        </form>
    );
};

export default EditControls;
