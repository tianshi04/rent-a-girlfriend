from dataclasses import dataclass

@dataclass(frozen=True)
class Location:
    city: str

    def __post_init__(self):
        if not isinstance(self.city, str):
            raise TypeError("City must be a string")
        
        # Standardize city names
        clean_city = self.city.strip()
        if not clean_city:
            raise ValueError("City name cannot be empty")
            
        # BR-19: City must belong to the active cities of the companion
        object.__setattr__(self, 'city', clean_city)

    def __str__(self) -> str:
        return self.city

