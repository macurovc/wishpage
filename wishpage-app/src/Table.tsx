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

const Table: React.FC<TableProps> = ({ items, setItems, person, category }) => {
    const filteredItems = items.filter(item => item.category === category && item.person === person)
    const [isDialogOpen, setIsDialogOpen] = useState(false)
    const [chosenItem, setChosenItem] = useState<Item>(NullItem)
    if (filteredItems.length == 0) {
        return undefined
    }

    const handleReserve = (item: Item) => {
        setChosenItem(item)
        setIsDialogOpen(true);
    };

    const handleReserveConfirm = async () => {
        const count = await request("PUT", `reserve/${chosenItem.id}`)
        if (count !== undefined) {
            setItems(items.map(item => item.id === chosenItem.id ? { ...item, count: item.count - 1 } : item))
        }

        console.log("Action confirmed");
        setIsDialogOpen(false);
    };

    const handleReserveCancel = () => {
        console.log("Action cancelled");
        setIsDialogOpen(false);
    };

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
                            Reserve
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
                                {row.count}
                            </td>
                            <td>
                                {row.count > 0 && <button onClick={() => handleReserve(row)}>
                                    Reserve 1
                                </button>}
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table >
            <ConfirmDialog
                isOpen={isDialogOpen}
                message={`Are you sure that you want to reserve "${chosenItem.name}"?`}
                onConfirm={handleReserveConfirm}
                onCancel={handleReserveCancel}
            />
        </div>
    );
};

export default Table;