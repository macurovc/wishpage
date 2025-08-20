import { useEffect } from 'react';
import Wishlist from './components/Wishlist';
import EditControls from './components/EditControls';
import ConfirmationDialog from './components/ConfirmationDialog';
import PasswordDialog from './components/PasswordDialog';
import { useStore } from './store';
import { Item, FamilyMember } from './types';

interface GroupedItems {
    [familyName: string]: Item[];
}

function App() {
    const {
        items,
        familyMembers,
        isEditMode,
        error,
        theme,
        selectedFamilyMemberName,
        confirmation,
        isPasswordDialogOpen,
        passwordError,
        hideConfirmation,
        showPasswordDialog,
        hidePasswordDialog,
        fetchItems,
        fetchFamilyMembers,
        addItem,
        reserveItem,
        deleteItem,
        addFamilyMember,
        deleteFamilyMember,
        toggleEditMode,
        setTheme,
        selectFamilyMember,
    } = useStore();

    useEffect(() => {
        fetchItems();
        fetchFamilyMembers();
    }, [fetchItems, fetchFamilyMembers]);

    useEffect(() => {
        const preferDarkMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
        setTheme(preferDarkMediaQuery.matches ? 'dark' : 'light');

        const handleThemeChange = (event: MediaQueryListEvent) => {
            const newTheme = event.matches ? 'dark' : 'light';
            setTheme(newTheme);
        };

        preferDarkMediaQuery.addEventListener('change', handleThemeChange);

        return () => {
            preferDarkMediaQuery.removeEventListener('change', handleThemeChange);
        };
    }, [setTheme]);

    const groupedItems: GroupedItems = {};
    items.sort((a: Item, b: Item) => (a.price || 0) - (b.price || 0));

    familyMembers.forEach((member: FamilyMember) => {
        groupedItems[member.name] = items.filter((item: Item) => item.family_member_id === member.id);
    });

    const familyFilteredGroupedItems = selectedFamilyMemberName
        ? { [selectedFamilyMemberName]: groupedItems[selectedFamilyMemberName] || [] }
        : groupedItems;

    return (
        <div className="App" data-theme={theme}>
            <div className="app-header">
                <h1>🎁 Family Wishlist</h1>
            </div>

            {error && <p className="error-message">Error: {error}</p>}

            <Wishlist
                groupedItems={familyFilteredGroupedItems}
                familyMembers={familyMembers}
                onReserve={reserveItem}
                isEditMode={isEditMode}
                onDelete={deleteItem}
                onSelectFamilyMember={selectFamilyMember}
                selectedFamilyMember={selectedFamilyMemberName}
            />

            {isEditMode &&
                <div className="edit-mode-controls">
                    <EditControls
                        onAddItem={addItem}
                        familyMembers={familyMembers}
                        onAddFamilyMember={addFamilyMember}
                        onDeleteFamilyMember={deleteFamilyMember}
                    />
                </div>
            }

            <div className="edit-button-container">
                {!isEditMode ? (
                    <button type="button" className="edit-mode-toggle" onClick={showPasswordDialog}>Edit ✍️</button>
                ) : (
                    <button type="button" className="edit-mode-toggle" onClick={() => toggleEditMode("")}>Exit Edit Mode 🔒</button>
                )}
            </div>

            <PasswordDialog
                isOpen={isPasswordDialogOpen}
                onConfirm={toggleEditMode}
                onCancel={hidePasswordDialog}
                error={passwordError}
            />

            <ConfirmationDialog
                isOpen={confirmation.isOpen}
                title={confirmation.title}
                message={confirmation.message}
                confirmText={confirmation.confirmText}
                onConfirm={() => {
                    if (confirmation.onConfirm) {
                        confirmation.onConfirm();
                    }
                    hideConfirmation();
                }}
                onCancel={hideConfirmation}
            />
        </div>
    );
}

export default App;
