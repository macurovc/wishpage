export interface Item {
  id: number;
  family_member_id: number;
  family_member_name: string;
  name: string;
  link?: string;
  price?: number | null;
  reserved: boolean; 
}

export interface FamilyMember {
  id: number;
  name: string;
}
