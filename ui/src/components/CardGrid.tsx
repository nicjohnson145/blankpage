import {type Metadata} from "../gen/blankpage/v1/struct_pb";
import Grid from "@mui/material/Grid";
import {BookCard} from "./BookCard";

type CardGridProps = {
    items: Metadata[];
};

function CardGrid({items}: CardGridProps) {
    return (
        <>
            <Grid container spacing={3} sx={{ justifyContent: 'flex-start' }}>
                {items.map((item) => {
                    return (
                        <BookCard key={item.id} metadata={item} />
                    )
                })}
            </Grid>
        </>
    )
}

export {
    CardGrid
}
