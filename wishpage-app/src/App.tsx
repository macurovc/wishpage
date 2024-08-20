import { useEffect, useState } from 'react'
import './App.css'
import Table from './Table'
import ItemForm, { existingCategories, Item } from './Item'
import AdminTable from './AdminTable'
import { Login } from './Login'
import { request } from './request'

function App() {
  const [isAdminMode, setIsAdminMode] = useState(false)
  const [items, setItems] = useState<Item[]>([]);

  useEffect(() => {
    fetchItems();
  }, []);

  const fetchItems = async () => {
    try {
      const items = await request("GET", "items", undefined, true)
      setItems(items);
    } catch (error) {
      console.error('There was a problem fetching the items:', error);
    }
  };

  const [category, setCategory] = useState<string>('')
  const people = [... new Set(items.map(item => item.person))].sort()
  const displayEditButton = category.length > 0

  return (
    <div>
      <h1>Wishlist</h1>
      {isAdminMode && <ItemForm fetchItems={fetchItems} people={people} />}
      <h4>What are you looking for?</h4>
      <div>
        {existingCategories.map(cat => (
          <button key={`${cat.name}_button`} onClick={() => setCategory(cat.name)} style={{
            backgroundColor: category === cat.name ? 'ButtonFace' : 'Canvas',
            margin: '5px',
            padding: '10px',
            fontSize: '18px',
          }}>{cat.symbol} {cat.name}</button>
        )
        )}
      </div>
      {people.map(person => {
        if (isAdminMode) {
          return <AdminTable key={`${person}_table`} items={items} setItems={setItems} person={person} category={category} />
        }
        return <Table key={`${person}_table`} items={items} setItems={setItems} person={person} category={category} />
      })}
      {Login({ isAdminMode, setIsAdminMode, displayEditButton })}
      <div style={{ marginTop: "100px", fontSize: "0.8em" }}>
        <a href='https://github.com/macurovc/wishpage' target="_blank" rel="noopener noreferrer">
          Github Project</a>
      </div>
    </div>
  )
}

export default App
