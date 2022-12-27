import "./style.css"
const Card = ({ lable, data }) => {

    return (
        <div className="main" >
            <h1 className="lable">
                {lable}
            </h1>
            <h1 className="data">
                {data}
            </h1>
        </div>
    )
}
export default Card;