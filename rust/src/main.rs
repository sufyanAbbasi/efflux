use efflux::{RenderableSocketData, Position};

fn main() {
    let mut req = RenderableSocketData::default();
    req.id = "my-id".to_owned();
    req.visible = true;

    let mut position = Position::default();
    position.x = 0;
    position.y = 1;
    position.z = 2;

    req.position = Some(position).into();
    println!("{}", req);
}
