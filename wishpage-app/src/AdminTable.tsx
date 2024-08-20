import { useState } from "react";
import { Item, labelForItem, NullItem } from "./Item";
import { ConfirmDialog } from "./ConfirmDialog";
import { request } from "./request";

interface TableProps {
    items: Item[]
    setItems: React.Dispatch<React.SetStateAction<Item[]>>
    person: string
    category: string
}

const AdminTable: React.FC<TableProps> = ({ items, setItems, person, category }) => {
    const filteredItems = items.filter(item => item.category === category && item.person === person)
    const [isDialogOpen, setIsDialogOpen] = useState(false)
    const [chosenItem, setChosenItem] = useState<Item>(NullItem)
    if (filteredItems.length == 0) {
        return undefined
    }

    const handleDelete = (item: Item) => {
        setChosenItem(item)
        setIsDialogOpen(true);
    };

    const handleDeleteConfirm = async () => {
        const token = localStorage.getItem("token")
        if (!token) {
            throw Error("cannot get the token from the local storage")
        }
        await request("DELETE", `admin/delete/${chosenItem.id}`, undefined, false, token)
        setItems(items.filter(item => item.id != chosenItem.id))

        console.log("Action confirmed");
        setIsDialogOpen(false);
    };

    const handleDeleteCancel = () => {
        console.log("Action cancelled");
        setIsDialogOpen(false);
    };

    const handleChangeCount = async (itemId: number, count: number) => {
        const token = localStorage.getItem("token")
        if (!token) {
            throw Error("cannot get the token from the local storage")
        }
        const item = { id: itemId, count: count }
        await request("PUT", `admin/update/${itemId}`, item, false, token)
        setItems(items.map(item => item.id === itemId ? { ...item, count: count } : item))
    }

    return (
        <div>
            <h2>{person}</h2>
            <table style={{ borderCollapse: 'collapse', width: '100%' }}>
                <thead>
                    <tr>
                        <th >
                            Item
                        </th>
                        <th>Amount</th>
                        <th>
                            Remove
                        </th>
                    </tr>
                </thead>
                <tbody>
                    {filteredItems.map((row) => (
                        <tr
                            key={row.id}
                        >
                            <td
                                style={{
                                    textDecoration: row.count === 0 ? 'line-through' : 'inherit',
                                }}
                            >{labelForItem(row)}</td>
                            <td>
                                {<input type="number" style={{ width: "40px" }} value={row.count} min={0} step={1} onChange={(event: React.ChangeEvent<HTMLInputElement>) => {
                                    const value = parseInt(event.target.value, 10)
                                    handleChangeCount(row.id, value)
                                }} />}
                            </td>
                            <td><button onClick={() => handleDelete(row)}>❌</button></td>
                        </tr>
                    ))}
                </tbody>
            </table >
            <ConfirmDialog
                isOpen={isDialogOpen}
                message={`Are you sure that you want to DELETE "${chosenItem.name}"?`}
                onConfirm={handleDeleteConfirm}
                onCancel={handleDeleteCancel}
            />
        </div>
    );
};

export default AdminTable;