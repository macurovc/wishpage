import React, { useState } from 'react';
import { request } from './request';
import { MessageDialog } from './ConfirmDialog';

export interface Item {
    id: number;
    name: string;
    person: string;
    link: string;
    price: number;
    count: number;
    category: string;
}

// eslint-disable-next-line react-refresh/only-export-components
export const NullItem: Item = {
    id: -1,
    name: "NULL",
    person: "NULL",
    link: "NULL",
    price: -1,
    count: -1,
    category: "NULL",
}

export interface Category {
    symbol: string;
    name: string;
}

// eslint-disable-next-line react-refresh/only-export-components
export const existingCategories: Array<Category> = [
    { name: "Shared Experience", symbol: "🍺" },
    { name: "Specific Item", symbol: "🎁" },
    { name: "Buyer's Choice", symbol: "💡" },
]

// eslint-disable-next-line react-refresh/only-export-components
export const labelForItem = (item: Item): JSX.Element => {
    let label = item.name
    if (item.price > 0) {
        label += ` (${item.price}€)`
    }
    if (item.link.length > 0) {
        return <a href={item.link} target="_blank" rel="noopener noreferrer">{label}</a>
    }
    return <div>{label}</div>
}

interface ItemFormProps {
    fetchItems: () => Promise<void>
    people: string[]
}

const ItemForm: React.FC<ItemFormProps> = ({ fetchItems, people }) => {
    const [isConfirmDialogOpen, setIsConfirmDialogOpen] = useState(false)
    const [item, setItem] = useState<Item>({
        id: -1,
        name: '',
        person: '',
        link: '',
        price: 0,
        count: 1,
        category: '',
    });
    const [person, setPerson] = useState('')

    const otherPerson = "> New Person..."
    const peopleOptions = [...people, otherPerson]

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
        const { name, value } = e.target;
        if (name == 'person') {
            setPerson(value)
        }
        if (name == 'other') {
            setItem(prevItem => ({ ...prevItem, ['person']: value }))
        } else {
            setItem(prevItem => ({
                ...prevItem,
                [name]: name === 'count' || name === 'price' ? parseInt(value) : value,
            }));
        }
    };

    const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const token = localStorage.getItem("token")
        if (!token) {
            throw Error("cannot get the token from the local storage")
        }
        console.log('Submitted item:', item);
        await request("POST", "admin/insert", item, false, token)
        setIsConfirmDialogOpen(true)
        await fetchItems()
    };

    return (
        <div>
            <h4>Create a new item</h4>
            <form onSubmit={handleSubmit}>
                <div className='form-grid'>
                    <div>
                        <label htmlFor="person">Person: </label>
                        <select
                            id="person"
                            name="person"
                            value={person}
                            onChange={handleChange}
                            required
                        >
                            <option value="">Select a person</option>
                            {peopleOptions.map(person => (
                                <option key={person} value={person}>
                                    {person}
                                </option>
                            ))}
                        </select>
                    </div>
                    {person == otherPerson && <div>
                        <label htmlFor="other">New person name: </label>
                        <input
                            type="text"
                            id="other"
                            name="other"
                            value={item.person}
                            onChange={handleChange}
                        />
                    </div>}
                    <div>
                        <label htmlFor="category">Category: </label>
                        <select
                            id="category"
                            name="category"
                            value={item.category}
                            onChange={handleChange}
                            required
                        >
                            <option value="">Select a category</option>
                            {existingCategories.map(category => (
                                <option key={category.name} value={category.name}>
                                    {category.name}
                                </option>
                            ))}
                        </select>
                    </div>
                    <div>
                        <label htmlFor="name">Name: </label>
                        <input
                            type="text"
                            id="name"
                            name="name"
                            value={item.name}
                            onChange={handleChange}
                            required
                        />
                    </div>
                    <div>
                        <label htmlFor="link">Link (optional): </label>
                        <input
                            type="url"
                            id="link"
                            name="link"
                            value={item.link}
                            onChange={handleChange}
                        />
                    </div>
                    <div>
                        <label htmlFor="price">Price € (optional): </label>
                        <input
                            type="number"
                            id="price"
                            name="price"
                            value={item.price}
                            onChange={handleChange}
                            step="1"
                            min={0}
                        />
                    </div>
                    <div>
                        <label htmlFor="count">Amount: </label>
                        <input
                            type="number"
                            id="count"
                            name="count"
                            value={item.count}
                            onChange={handleChange}
                            step="1"
                            min={1}
                        />
                    </div>
                </div>
                <div>
                    <button style={{ marginTop: "20px", color: "ButtonText", backgroundColor: "ButtonFace" }}
                        type="submit">Create Item</button>
                </div>
            </form>
            <MessageDialog
                isOpen={isConfirmDialogOpen}
                message={`The item "${item.name}" was successfully created`}
                onConfirm={() => setIsConfirmDialogOpen(false)}
            />
        </div>
    );
};

export default ItemForm;
